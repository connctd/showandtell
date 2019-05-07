package showandtell

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderSlides(t *testing.T) {
	pres := &Presentation{
		Name: "foo",
	}

	out, err := RenderIndex(pres, "./test_slides")
	require.NoError(t, err)
	fmt.Printf("Output:\n%s\n", string(out))
}
