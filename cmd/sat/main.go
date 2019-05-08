package main

import (
	"os"

	"github.com/connctd/showandtell"
	"github.com/urfave/cli"
)

var (
	slideFolder string
)

func main() {
	app := cli.NewApp()
	app.Name = "sat"
	app.HelpName = "Show And Tell"
	app.Version = showandtell.Version
	app.Description = "Render and serve reveal.js based presentations"
	app.EnableBashCompletion = true
	app.Commands = []cli.Command{renderCommand}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "slides",
			Value:       "slides",
			Usage:       "The location of the slides to render",
			Destination: &slideFolder,
		},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
	destDir := "./test_out"

	if err := showandtell.EmitRevealJS(destDir); err != nil {
		panic(err)
	}
}
