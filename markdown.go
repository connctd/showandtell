package showandtell

import (
	"html/template"

	blackfriday "gopkg.in/russross/blackfriday.v2"
)

func init() {
	RegisterSlideFormat("md", &MarkdownSlideParser{})
}

const mardownExtensions = blackfriday.NoIntraEmphasis | blackfriday.Tables | blackfriday.FencedCode |
	blackfriday.Strikethrough | blackfriday.SpaceHeadings | blackfriday.HeadingIDs |
	blackfriday.BackslashLineBreak | blackfriday.DefinitionLists

type MarkdownSlideParser struct{}

func (m *MarkdownSlideParser) ParseSlide(ctx *SlideContext, input []byte) (content template.HTML, err error) {
	out := blackfriday.Run(input,
		blackfriday.WithExtensions(
			mardownExtensions,
		),
	)
	return template.HTML(out), nil
}
