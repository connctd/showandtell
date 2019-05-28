package showandtell

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var showOutput = false

func TestRenderSlides(t *testing.T) {
	pres := &Presentation{
		Name: "foo",
	}

	out, err := RenderIndex(pres, "./test_slides")
	require.NoError(t, err)
	if showOutput {
		fmt.Printf("Output:\n%s\n", string(out))
	}
}

func TestParseFrontMatter(t *testing.T) {
	for _, data := range []struct {
		in           string
		expectedFm   string
		expectedBody string
	}{
		{
			`+++
Test
+++`,
			`
Test
`,
			``,
		},
		{
			`foobar`,
			``,
			`foobar`,
		},
		{
			`+++
Test`,
			`
Test`,
			``,
		},
	} {
		fm, body := parseFrontMatter([]byte(data.in))
		assert.Equal(t, []byte(data.expectedFm), fm)
		assert.Equal(t, []byte(data.expectedBody), body)
	}
}
