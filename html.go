package showandtell

import (
	"html/template"
)

type HTMLSlideParser struct{}

func init() {
	RegisterSlideFormat("html", &HTMLSlideParser{})
}

func (h *HTMLSlideParser) ParseSlide(ctx *SlideContext, input []byte) (content template.HTML, err error) {
	return template.HTML(input), nil
}
