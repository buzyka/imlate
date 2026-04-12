package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReaderTemplate_ShowsFullLogoWithoutBackgroundCropping(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "website", "reader.html"))
	require.NoError(t, err)

	template := string(data)
	assert.Contains(t, template, `<img id="logo-image"`)
	assert.Contains(t, template, `object-fit: contain`)
	assert.NotContains(t, template, `background-size: cover`)
	assert.False(t, strings.Contains(template, `logoBlock.style.backgroundImage = "url('`))
}
