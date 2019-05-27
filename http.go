package showandtell

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type indexHandler struct {
	indexBytes []byte
}

func (i *indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write(i.indexBytes)
}

type PresentationServer struct {
	slideDir   string
	pres       *Presentation
	ctx        context.Context
	httpServer *http.Server
	indexBytes []byte

	indexLock *sync.Mutex
}

func NewPresentationServer(ctx context.Context, pres *Presentation, slideDir, addr string) (*PresentationServer, error) {
	server := &http.Server{
		Addr: addr,
	}

	p := &PresentationServer{
		ctx:        ctx,
		pres:       pres,
		slideDir:   slideDir,
		httpServer: server,
		indexLock:  &sync.Mutex{},
	}

	if err := p.Rerender(); err != nil {
		return nil, err
	}

	mux := ServeRevealJS()
	mux.Handle("/", http.HandlerFunc(p.serveIndex))
	server.Handler = mux

	return p, nil
}

func (p *PresentationServer) serveIndex(w http.ResponseWriter, r *http.Request) {
	p.indexLock.Lock()
	defer p.indexLock.Unlock()
	w.Write(p.indexBytes)
}

func (p *PresentationServer) Rerender() (err error) {
	p.indexLock.Lock()
	defer p.indexLock.Unlock()
	p.indexBytes, err = RenderIndex(p.pres, p.slideDir)
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

func ServePresentation(ctx context.Context, pres *Presentation, slideDir, addr string, rerenderChan chan bool) (*http.Server, error) {
	server := &http.Server{
		Addr: addr,
	}

	indexBytes, err := RenderIndex(pres, slideDir)
	if err != nil {
		return nil, err
	}
	index := &indexHandler{indexBytes}

	mux := ServeRevealJS()
	mux.Handle("/", index)

	server.Handler = mux

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case val := <-rerenderChan:
				if val {
					indexBytes, err := RenderIndex(pres, slideDir)
					if err != nil {
						logrus.WithError(err).Error("Failed to rerender the presentation for live reload")
						continue
					}
					index.indexBytes = indexBytes
				}
			}
		}
	}()

	return server, nil
}
