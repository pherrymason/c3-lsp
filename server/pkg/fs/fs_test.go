package fs

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
)

func TestConvertPathToURI(t *testing.T) {
	path := `D:\projects\c3-lsp\assets\c3-demo\foobar\foo.c3`

	uri := ConvertPathToURI(path, option.None[string]())

	assert.Equal(t, "file:///D:/projects/c3-lsp/assets/c3-demo/foobar/foo.c3", uri)
}
