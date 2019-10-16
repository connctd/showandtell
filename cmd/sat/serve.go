package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/connctd/showandtell"
	"github.com/fsnotify/fsnotify"
	"github.com/urfave/cli"
)

var httpAddr string

var serveCommand = cli.Command{
	Name:        "serve",
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
		cctx := context.Background()
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		if err := watcher.Add(slideFolder); err != nil {
			return err
		}
		subFiles, err := ioutil.ReadDir(slideFolder)
		if err != nil {
			return err
		}
		for _, fi := range subFiles {
			// Add subfolder to watcher, because the watcher is not recursive
			if fi.IsDir() {
				if err := watcher.Add(filepath.Join(slideFolder, fi.Name())); err != nil {
					return err
				}
			}
		}

		var server *showandtell.PresentationServer

		server, err = showandtell.NewPresentationServer(cctx, presentation, slideFolder, httpAddr)
		if err != nil {
			return
		}
		fmt.Printf("Serving presentation on %s\n", httpAddr)
		server.Run()

		go func() {
			for {
				select {
				case <-cctx.Done():
					return
				case evt := <-watcher.Events:

					switch evt.Op {
					case fsnotify.Write:
						fmt.Printf("File %s changed, rerendering...\n", evt.Name)
						server.Rerender()
					case fsnotify.Create:
						fmt.Printf("File %s created, rerendering...\n", evt.Name)
						if isDirectory(evt.Name) {
							watcher.Add(evt.Name)
						}
						server.Rerender()
					case fsnotify.Remove:
						fmt.Printf("File %s deleted, rerendering...\n", evt.Name)
						server.Rerender()
					case fsnotify.Rename:
						fmt.Printf("File %s renamed, rerendering...\n", evt.Name)
						server.Rerender()
					default:
						continue
					}
				}
			}
		}()

		select {
		case <-c:
			server.Close()
		}
		return
	},
}

func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}
