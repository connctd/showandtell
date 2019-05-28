package showandtell

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	bus "github.com/vardius/message-bus"
)

var (
	writeWait = 10 * time.Second

	busQueueSize = 100
)

func messageBusClientF(ctx context.Context,
	logger logrus.FieldLogger,
	cancel context.CancelFunc,
	ws *websocket.Conn,
	messageBus bus.MessageBus) {
	logger = logger.WithFields(logrus.Fields{
		"remoteAddr": ws.RemoteAddr().String(),
	})
	logger.Debug("Handling client connection")
	defer cancel()

	subscriptionHandler := func(topic string) func(json.RawMessage) {
		return func(value json.RawMessage) {
			msg := &WebSocketBusMessage{
				Value: value,
				Topic: topic,
				Type:  "message",
			}
			if err := ws.WriteJSON(msg); err != nil {
				logger.WithError(err).Error("Failed to write to client, closing connection")
				ws.Close()
				cancel()
			}
		}
	}

	go func() {
		for {
			msg := &WebSocketBusMessage{}
			err := ws.ReadJSON(msg)
			if err != nil {
				logger.WithError(err).Error("Failed to read message from client, closing connection")
				ws.Close()
				return
			}
			logger.Debug("Received msg from client")

			switch msg.Type {
			case "subscribe":
				logger.Debug("Subscribing client")
				messageBus.Subscribe(msg.Topic, subscriptionHandler(msg.Topic))
			case "unsubscribe":
				logger.Debug("Unsubscribing client")
				// FIXME, this won't likely work.
				messageBus.Unsubscribe(msg.Topic, subscriptionHandler)
			case "publish":
				logger.Debug("Publishing message")
				messageBus.Publish(msg.Topic, msg.Value)
			}
		}
	}()
	select {
	case <-ctx.Done():
		logger.Debug("Context done, closing connection")
		ws.Close()
		return
	}
}

type WebSocketBusMessage struct {
	Type  string          `json:"type"`
	Topic string          `json:"topic"`
	Value json.RawMessage `json:"value"`
}

type PresentationServer struct {
	slideDir        string
	pres            *Presentation
	ctx             context.Context
	httpServer      *http.Server
	indexBytes      []byte
	wsUpgrader      websocket.Upgrader
	livereloadConns []*websocket.Conn
	centralBus      bus.MessageBus
	logger          logrus.FieldLogger

	indexLock *sync.Mutex
}

func NewPresentationServer(ctx context.Context, pres *Presentation, slideDir, addr string) (*PresentationServer, error) {
	server := &http.Server{
		Addr: addr,
	}

	p := &PresentationServer{
		ctx:             ctx,
		pres:            pres,
		slideDir:        slideDir,
		httpServer:      server,
		indexLock:       &sync.Mutex{},
		wsUpgrader:      websocket.Upgrader{},
		livereloadConns: make([]*websocket.Conn, 0, 100),
		centralBus:      bus.New(busQueueSize),
		logger:          logrus.WithField("component", "PresentationServer"),
	}

	if err := p.Rerender(); err != nil {
		return nil, err
	}

	mux := ServeRevealJS()
	mux.Handle("/", http.HandlerFunc(p.serveIndex))
	mux.Handle("/livereload", http.HandlerFunc(p.livereloadHandler))
	mux.Handle("/messagebus", http.HandlerFunc(p.messagebusHandler))
	server.Handler = mux

	return p, nil
}

func (p *PresentationServer) serveIndex(w http.ResponseWriter, r *http.Request) {
	p.indexLock.Lock()
	defer p.indexLock.Unlock()
	w.Write(p.indexBytes)
}

func (p *PresentationServer) livereloadHandler(w http.ResponseWriter, r *http.Request) {
	logger := p.logger.WithFields(logrus.Fields{
		"remoteAddr": r.RemoteAddr,
		"url":        r.URL.String(),
	})
	ws, err := p.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.WithError(err).Error("Failed to start websocket connection")
		return
	}

	p.livereloadConns = append(p.livereloadConns, ws)
	ctx, cancel := context.WithCancel(p.ctx)
	go ping(ctx, cancel, ws)
}

func (p *PresentationServer) messagebusHandler(w http.ResponseWriter, r *http.Request) {
	logger := p.logger.WithFields(logrus.Fields{
		"remoteAddr": r.RemoteAddr,
		"url":        r.URL.String(),
	})
	ws, err := p.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.WithError(err).Error("Failed to start websocket connection")
		return
	}
	ctx, cancel := context.WithCancel(p.ctx)
	logger = logger.WithField("messagebus", "websocket")
	go messageBusClientF(ctx, logger, cancel, ws, p.centralBus)
}

func ping(ctx context.Context, cancel context.CancelFunc, ws *websocket.Conn) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				log.Println("ping:", err)
				return
			}
		}
	}
}

func (p *PresentationServer) Rerender() (err error) {
	p.indexLock.Lock()
	p.indexBytes, err = RenderIndex(p.pres, p.slideDir)
	p.indexLock.Unlock()
	go func() {
		for _, ws := range p.livereloadConns {
			logger := p.logger.WithFields(logrus.Fields{
				"remoteAddr": ws.RemoteAddr().String(),
			})
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.TextMessage, []byte(`Reload`)); err != nil {
				logger.WithError(err).Warn("Failed to write reload message to client, closing connection")
				// TODO remove connection
				ws.Close()
			}
		}
	}()
	return
}

func (p *PresentationServer) Close() error {
	ctx, cancel := context.WithTimeout(p.ctx, time.Second*15)
	defer cancel()
	return p.httpServer.Shutdown(ctx)
}

func (p *PresentationServer) Run() {
	go func() {
		p.httpServer.ListenAndServe()
	}()
}
