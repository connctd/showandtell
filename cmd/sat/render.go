package main

import (
	"os"
	"path/filepath"

	"github.com/connctd/showandtell"
	"github.com/urfave/cli"
)

var defaultDistDir = "./dist"

var renderCommand = cli.Command{
	Name:    "render",
	Aliases: []string{"build", "r", "b"},
	Usage:   "Render the presentation into the dist dir",
	Action: func(ctx *cli.Context) error {
		distDir := ctx.Args().First()
		if distDir == "" {
			distDir = defaultDistDir
		}
		if err := showandtell.EmitRevealJS(distDir); err != nil {
			return err
		}
		indexPath := filepath.Join(distDir, "index.html")
		indexFile, err := os.Create(indexPath)
		if err != nil {
			return err
		}
		defer indexFile.Close()

		indexBytes, err := showandtell.RenderIndex(&showandtell.Presentation{}, slideFolder)
		if err != nil {
			return err
		}
		_, err = indexFile.Write(indexBytes)
		return err
	},
}
