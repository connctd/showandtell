package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/connctd/showandtell"
	"github.com/urfave/cli"
)

var httpAddr string

var serveCommand = cli.Command{
	Name:        "server",
	Aliases:     []string{"s"},
	Description: "Serve the presentation on a webserver",
	Usage:       "serve [--addr :8080]",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:        "addr",
			Usage:       "Specify the address to listen on",
			Value:       ":8080",
			Destination: &httpAddr,
		},
	},
	Action: func(ctx *cli.Context) (err error) {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			var server *http.Server
			server, err = showandtell.ServePresentation(presentation, slideFolder, httpAddr)
			if err != nil {
				return
			}

			err = server.ListenAndServe()
		}()

		select {
		case <-c:

		}
		return
	},
}
