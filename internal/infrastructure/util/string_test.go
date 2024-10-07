package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomString(t *testing.T) {
	assert.Equal(t, 10, len(RandomString(10)))
}

func TestRandomStringWithCharset(t *testing.T) {
	str := RandomStringWithCharset(10, "ab")
	assert.Equal(t, 10, len(str))
	for _, char := range str {
		assert.True(t, strings.ContainsRune("ab", char), "String should contains only 'a' and 'b'")
	}
}
