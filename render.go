package showandtell

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"strings"

	blackfriday "gopkg.in/russross/blackfriday.v2"
	"gopkg.in/yaml.v2"
)

var Version = "undefined"

var frontMatterDelimiter = []byte(`+++`)

var mainTmpl = `[[define "main" ]] [[ template "base" . ]] [[ end ]]`

var baseTmpl = `
[[ define "base" ]]
<html>
	<head>
		<link rel="stylesheet" href="css/reveal.css">
		<link rel="stylesheet" href="css/theme/white.css">
	</head>
	<body>
		<div class="reveal">
			<div class="slides">
				[[ range .Slides ]]
					[[ template "comboSlide" . ]]
				[[ end ]]
			</div>
		</div>
		[[ block "js" . ]]
		<script src="js/reveal.js"></script>
		<script>
			Reveal.initialize();
		</script>
		[[ end ]]
	</body>
</html>
[[ end ]]
`

var comboSlide = `
[[ define "comboSlide" ]]
[[ if .SubSlides ]]
[[ template "subSlides" . ]]
[[ else ]]
[[ template "slide" . ]]
[[ end ]]
[[ end ]]
`

var subSlideTmpl = `
[[ define "subSlides" ]]
<section id="[[ .SectionID ]]" class="chapter">
[[ range .SubSlides ]]
	[[ template "slide" . ]]
[[ end ]]
</section>
[[ end ]]
`

var slideTmpl = `
[[ define "slide" ]]
<section id="[[ .SectionID ]]" data-has-notes="[[ .HasNotes ]]" class="slide">
[[ .Content ]]
[[ if .HasNotes ]]
<aside class="notes">
[[.Notes]]
</aside>
[[ end ]]
</section>
[[ end ]]
`

var slideParsers = map[string]SlideParser{}

func RegisterSlideFormat(ext string, parser SlideParser) {
	slideParsers[ext] = parser
}

type Slide struct {
	Content    template.HTML
	SourceFile string
	SubSlides  []*Slide
	Notes      template.HTML `yaml:"notes"`
	SectionID  string
}

func (s *Slide) HasNotes() bool {
	return len(s.Notes) > 0
}

type SlideContext struct {
	Slide
	Presentation
}

type Presentation struct {
	Name        string
	Description string
	Slides      []*Slide
}

type SlideParser interface {
	ParseSlide(ctx *SlideContext, input []byte) (template.HTML, error)
}

func DefaultRenderer() *template.Template {
	var err error
	tmpl := template.New("main")
	// TODO add func map
	tmpl.Delims("[[", "]]")
	for _, tmplStr := range []string{mainTmpl, baseTmpl, slideTmpl, subSlideTmpl, comboSlide} {
		tmpl, err = tmpl.Parse(tmplStr)
		if err != nil {
			panic(err)
		}
	}

	return tmpl
}

func parseFrontMatter(in []byte) (fm []byte, content []byte) {
	if !bytes.HasPrefix(in, frontMatterDelimiter) {
		return nil, in
	}

	parts := bytes.SplitN(in, frontMatterDelimiter, 3)

	return parts[1], parts[2]
}

func generateSectionID(slidePath string) string {
	extension := filepath.Ext(slidePath)
	fileName := filepath.Base(slidePath)
	name := strings.TrimSuffix(fileName, extension)

	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "_")

	return id
}

func parseSlide(pres *Presentation, slidePath string) (s *Slide, err error) {
	extension := filepath.Ext(slidePath)
	extension = strings.TrimPrefix(extension, ".")
	buf, err := ioutil.ReadFile(slidePath)
	if err != nil {
		return nil, err
	}

	frontMatter, body := parseFrontMatter(buf)
	if err != nil {
		return nil, err
	}
	s = &Slide{}

	if len(frontMatter) > 0 {
		if err := yaml.Unmarshal(frontMatter, s); err != nil {
			return nil, err
		}
	}

	s.SourceFile = slidePath
	s.SectionID = generateSectionID(slidePath)

	if s.HasNotes() {
		// Notes are in Markdown, so we render it to HTML
		s.Notes = template.HTML(blackfriday.Run([]byte(s.Notes), blackfriday.WithExtensions(
			mardownExtensions,
		)))
	}

	slideCtx := &SlideContext{
		*s,
		*pres,
	}

	tmpl := DefaultRenderer()

	tmpl, err = tmpl.Parse(string(body))
	if err != nil {
		return
	}
	tmplBuf := &bytes.Buffer{}
	if err = tmpl.Execute(tmplBuf, slideCtx); err != nil {
		return
	}

	if parser, exists := slideParsers[extension]; exists {
		s.Content, err = parser.ParseSlide(slideCtx, []byte(body))
		if err != nil {
			return nil, err
		}
	} else {
		err = fmt.Errorf("No matching slide parser for file type: %s", extension)
	}
	return
}

func renderSubSlides(pres *Presentation, slide *Slide) (template.HTML, error) {
	slideTmpl := DefaultRenderer()
	buf := &bytes.Buffer{}

	err := slideTmpl.ExecuteTemplate(buf, "subSlides", slide)
	return template.HTML(buf.String()), err
}

func parseSlideFolder(pres *Presentation, slideFolder string) (slides []*Slide, err error) {
	files, err := ioutil.ReadDir(slideFolder)
	if err != nil {
		return nil, err
	}
	for _, f := range files {

		slidePath := filepath.Join(slideFolder, f.Name())
		if f.IsDir() {
			subSlides, err := parseSlideFolder(pres, slidePath)
			if err != nil {
				return nil, err
			}
			id := generateSectionID(slidePath)
			for _, s := range subSlides {
				s.SectionID = id + "-" + s.SectionID
			}
			s := &Slide{
				SourceFile: slidePath,
				SubSlides:  subSlides,
				SectionID:  id,
			}
			slides = append(slides, s)
		} else {

			s, err := parseSlide(pres, slidePath)
			if err != nil {
				return nil, err
			}
			slides = append(slides, s)
		}
	}
	return slides, nil
}

func ParseSlides(pres *Presentation, slideFolder string) ([]*Slide, error) {
	return parseSlideFolder(pres, slideFolder)
}

func RenderIndex(pres *Presentation, slideFolder string) ([]byte, error) {
	slides, err := ParseSlides(pres, slideFolder)
	if err != nil {
		return nil, err
	}

	pres.Slides = slides

	tmpl := DefaultRenderer()
	buf := &bytes.Buffer{}
	err = tmpl.ExecuteTemplate(buf, "main", pres)
	return buf.Bytes(), err
}
