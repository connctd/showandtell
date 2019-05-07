package showandtell

import (
	"os"
	"path/filepath"

	"github.com/gobuffalo/packr/v2"
)

var (
	cssBox    = packr.New("css", "./dist_temp/reveal/css")
	libBox    = packr.New("lib", "./dist_temp/reveal/lib")
	jsBox     = packr.New("js", "./dist_temp/reveal/js")
	pluginBox = packr.New("plugin", "./dist_temp/reveal/plugin")

	revealBoxes = []*packr.Box{cssBox, libBox, jsBox, pluginBox}
)

func EmitRevealJS(destDir string) error {
	for _, b := range revealBoxes {
		destPath := filepath.Join(destDir, b.Name)
		if err := os.MkdirAll(destPath, 0777); err != nil {
			return nil
		}
		for _, f := range b.List() {
			fPath := filepath.Join(destPath, f)
			outDir := filepath.Dir(fPath)

			if outDir != destPath {
				if err := os.MkdirAll(outDir, 0777); err != nil {
					return err
				}
			}
			outFile, err := os.Create(fPath)
			if err != nil {
				return err
			}
			defer outFile.Close()
			data, err := b.Find(f)
			if err != nil {
				return err
			}
			if _, err := outFile.Write(data); err != nil {
				return err
			}
		}
	}
	return nil
}
