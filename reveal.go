package showandtell

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobuffalo/packr/v2"
)

var (
	cssBox    = packr.New("css", "./dist_temp/reveal/css")
	libBox    = packr.New("lib", "./dist_temp/reveal/lib")
	jsBox     = packr.New("js", "./dist_temp/reveal/js")
	pluginBox = packr.New("plugin", "./dist_temp/reveal/plugin")
	imagesBox = packr.New("images", "./dist_temp/reveal/img")

	revealBoxes = []*packr.Box{cssBox, libBox, jsBox, pluginBox, imagesBox}
)

func dirExists(dirPath string) bool {
	if _, err := os.Stat(dirPath); err != nil && os.IsNotExist(err) {
		return false
	} else if err != nil && os.IsExist(err) {
		return true
	}
	return true
}

func AddCustomFiles(baseDir string) error {
	if !dirExists(baseDir) {
		return nil
	}

	for _, b := range revealBoxes {
		dirPath := filepath.Join(baseDir, b.Name)
		if !dirExists(dirPath) {
			continue
		}

		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				relPath := strings.TrimPrefix(path, dirPath)
				relPath = strings.TrimPrefix(relPath, "/")
				fmt.Printf("Adding custom file %s to %s\n", relPath, b.Name)
				fileBytes, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}
				if err := b.AddBytes(relPath, fileBytes); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func ServeRevealJS() *http.ServeMux {
	mux := &http.ServeMux{}

	for _, b := range revealBoxes {
		path := "/" + b.Name + "/"
		mux.Handle(path, http.StripPrefix(path, http.FileServer(b)))
	}
	return mux
}

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
