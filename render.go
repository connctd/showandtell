package showandtell

import (
	"bytes"
	"encoding/json"
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
			Reveal.initialize([[.RevealConfig.ToJSON]]);
		</script>
		<script>
		function tryConnectToReload(address) {
			var conn;
			var url = window.location.host+"/livereload";
			if(window.location.protocol === "http:") {
				url = "ws://"+url;
			} else {
				url = "wss://"+url;
			}
			// This is a statically defined port on which the app is hosting the reload service.
			conn = new WebSocket(url);
		
			conn.onclose = function(evt) {
				// The reload endpoint hasn't been started yet, we are retrying in 2 seconds.
				setTimeout(() => tryConnectToReload(), 2000);
			};
		
			conn.onmessage = function(evt) {
				console.log("Refresh received!");
		
				// If we uncomment this line, then the page will refresh every time a message is received.
				location.reload()
			};
		}
		
		try {
			if (window["WebSocket"]) {
				tryConnectToReload();
			} else {
				console.log("Your browser does not support WebSocket, cannot connect to the reload service.");
			}
		} catch (ex) {
			console.log('Exception during connecting to reload:', ex);
		}
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
<section 
	class="slide"
	id="[[ .SectionID ]]" 
	data-has-notes="[[ .HasNotes ]]" 
	[[ if .Transition ]]data-transition="[[.Transition]]" [[if .TransitionSpeed]]data-transition-speed="[[.TransitionSpeed]]" [[end]][[end]]>
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

type RevealDependency struct {
	RelSrc string `json:"src"`
	Async  bool   `json:"async"`
}

type RevealConfiguration struct {
	Controls         *bool `json:"controls,omitempty"`
	ControlsTutorial *bool `json:"controlsTutorial,omitempty"`
	// Determines where controls appear, "edges" or "bottom-right"
	ControlsLayout *string `json:"controlsLayout,omitempty"`
	// Visibility rule for backwards navigation arrows; "faded", "hidden"
	// or "visible"
	ControlsBackArrows *string `json:"controlsBackArrows,omitempty"`
	Progress           *bool   `json:"progress,omitempty"`
	SlideNumber        *bool   `json:"slideNumber,omitempty"`
	Hash               *bool   `json:"hash,omitempty"`
	History            *bool   `json:"history,omitempty"`
	Keyboard           *bool   `json:"keyboard,omitempty"`
	Overview           *bool   `json:"overview,omitempty"`
	Center             *bool   `json:"center,omitempty"`
	Touch              *bool   `json:"touch,omitempty"`
	Loop               *bool   `json:"loop,omitempty"`
	Rtl                *bool   `json:"rtl,omitempty"`
	// See https://github.com/hakimel/reveal.js/#navigation-mode
	NavigdationMode    *string `json:"navigationMode,omitempty"`
	Shuffle            *bool   `json:"shuffle,omitempty"`
	Fragments          *bool   `json:"fragments,omitempty"`
	FragmentInURL      *bool   `json:"fragmentInURL,omitempty"`
	Embedded           *bool   `json:"embedded,omitempty"`
	Help               *bool   `json:"help,omitempty"`
	ShowNotes          *bool   `json:"showNotes,omitempty"`
	AutoPlayMedia      *bool   `json:"autoPlayMedia"`
	PreloadIFrames     *bool   `json:"preloadIframes"`
	AutoSlide          *uint64 `json:"autoSlide,omitempty"`
	AutoSlideStoppable *bool   `json:"autoSlideStoppable,omitempty"`
	DefaultTiming      *uint64 `json:"defaultTiming,omitempty"`
	MouseWheel         *bool   `json:"mouseWheel,omitempty"`
	HideInactiveCursor *bool   `json:"hideInactiveCursor,omitempty"`
	HideCursorTime     *uint64 `json:"hideCursorTime,omitempty"`
	HideAddressBar     *bool   `json:"hideAddressBar,omitempty"`
	PreviewLinks       *bool   `json:"previewLinks,omitempty"`
	// none/fade/slide/convex/concave/zoom
	Transition *string `json:"transition,omitempty"`
	// default/fast/slow
	TransitionSpeed         *string `json:"transitionSpeed,omitempty"`
	BackgroundTransition    *string `json:"backgroundTransition,omitempty"` // none/fade/slide/convex/concave/zoom
	ViewDistance            *uint64 `json:"viewDistance,omitempty"`
	ParallaxBackgroundImage *string `json:"parallaxBackgroundImage,omitempty"`
	ParallaxBackgroundSize  *string `json:"parallaxBackgroundSize,omitempty"`
	// TODO parallaxBackgroundHorizontal: null, parallaxBackgroundVertical: null,
	Display      *string             `json:"display,omitempty"`
	Dependencies []*RevealDependency `json:"dependencies"`
}

func (r *RevealConfiguration) ToJSON() template.JS {
	s, _ := json.Marshal(r)
	return template.JS(s)
}

func DefaultRevealConfig() *RevealConfiguration {
	return &RevealConfiguration{
		Controls: Bool(true),
		Progress: Bool(true),
		History:  Bool(true),
		Center:   Bool(true),

		Dependencies: []*RevealDependency{
			{
				RelSrc: "plugin/notes/notes.js",
				Async:  true,
			},
			{
				RelSrc: "plugin/zoom-js/zoom.js",
				Async:  true,
			},
			{
				RelSrc: "plugin/highlight/highlight.js",
				Async:  true,
			},
		},
	}
}

func Bool(in bool) *bool {
	return &in
}

type Slide struct {
	Content    template.HTML
	SourceFile string
	SubSlides  []*Slide
	SectionID  string

	Notes           template.HTML `yaml:"notes"`
	Transition      *string       `yaml:"transition"`
	TransitionSpeed *string       `yaml:"transitionSpeed"`
}

func (s *Slide) HasNotes() bool {
	return len(s.Notes) > 0
}

type SlideContext struct {
	Slide
	Presentation
}

type Presentation struct {
	Name         string               `yaml:"name"`
	Description  string               `yaml:"description"`
	Slides       []*Slide             `json:"-"`
	RevealConfig *RevealConfiguration `yaml:"reveal_config"`
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
		return []byte{}, in
	}

	parts := bytes.SplitN(in, frontMatterDelimiter, 3)
	if len(parts) < 3 {
		return parts[1], []byte{}
	}

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

func ParsePresentation(presPath string) (*Presentation, error) {
	buf, err := ioutil.ReadFile(presPath)
	if err != nil {
		return nil, err
	}
	pres := &Presentation{}
	err = yaml.Unmarshal(buf, pres)
	if err != nil {
		return nil, err
	}
	if pres.RevealConfig == nil {
		// TODO use a nice and sane default configuration
		pres.RevealConfig = DefaultRevealConfig()
	}
	return pres, nil
}
