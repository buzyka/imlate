package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileExistsWillFindFile(t *testing.T) {
	assert.True(t, FileExists("./filesystem.go"))
}

func TestFileExistsWithNotExistingFileWillReturnFalse(t *testing.T) {
	assert.False(t, FileExists("./some_file.txt"))
}

func TestFileExistsWithDirectoryNameWillReturnFalse(t *testing.T) {
	assert.False(t, FileExists("../util"))
}
