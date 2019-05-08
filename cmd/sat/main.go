package main

import (
	"os"

	"github.com/connctd/showandtell"
	"github.com/urfave/cli"
)

var (
	slideFolder string

	presentationPath string
	presentation     *showandtell.Presentation
)

func main() {
	app := cli.NewApp()
	app.Name = "sat"
	app.HelpName = "Show And Tell"
	app.Version = showandtell.Version
	app.Description = "Render and serve reveal.js based presentations"
	app.EnableBashCompletion = true
	app.Commands = []cli.Command{renderCommand, serveCommand}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "slides",
			Value:       "slides",
			Usage:       "The location of the slides to render",
			Destination: &slideFolder,
		},

		cli.StringFlag{
			Name:        "config",
			Value:       "./presentation.yaml",
			Usage:       "Specify an alternative location for presentation yaml",
			Destination: &presentationPath,
		},
	}
	app.Before = func(ctx *cli.Context) (err error) {
		presentation, err = showandtell.ParsePresentation(presentationPath)
		return err
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}
