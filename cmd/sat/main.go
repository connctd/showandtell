package main

import (
	"github.com/connctd/showandtell"
)

func main() {
	destDir := "./test_out"

	if err := showandtell.EmitRevealJS(destDir); err != nil {
		panic(err)
	}
}
