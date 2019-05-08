package showandtell

import (
	"net/http"
)

func serveIndex(indexBytes []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(indexBytes)
	})
}

func ServePresentation(pres *Presentation, slideDir, addr string) (*http.Server, error) {
	server := &http.Server{
		Addr: addr,
	}

	indexBytes, err := RenderIndex(pres, slideDir)
	if err != nil {
		return nil, err
	}

	mux := ServeRevealJS()
	mux.Handle("/", serveIndex(indexBytes))

	server.Handler = mux

	return server, nil
}
