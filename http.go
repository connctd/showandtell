package showandtell

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	writeWait = 10 * time.Second
)

type PresentationServer struct {
	slideDir        string
	pres            *Presentation
	ctx             context.Context
	httpServer      *http.Server
	indexBytes      []byte
	wsUpgrader      websocket.Upgrader
	livereloadConns []*websocket.Conn

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
	}

	if err := p.Rerender(); err != nil {
		return nil, err
	}

	mux := ServeRevealJS()
	mux.Handle("/", http.HandlerFunc(p.serveIndex))
	mux.Handle("/livereload", http.HandlerFunc(p.livereloadHandler))
	server.Handler = mux

	return p, nil
}

func (p *PresentationServer) serveIndex(w http.ResponseWriter, r *http.Request) {
	p.indexLock.Lock()
	defer p.indexLock.Unlock()
	w.Write(p.indexBytes)
}

func (p *PresentationServer) livereloadHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := p.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	p.livereloadConns = append(p.livereloadConns, ws)
	ctx, cancel := context.WithCancel(p.ctx)
	go ping(ctx, cancel, ws)
}

func ping(ctx context.Context, cancel context.CancelFunc, ws *websocket.Conn) {
	ticker := time.NewTicker(time.Second * 30)
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
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.TextMessage, []byte(`Reload`)); err != nil {
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
