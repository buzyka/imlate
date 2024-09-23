package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInArrayWillReturnTrue(t *testing.T) {
	assert.True(t, InArray("needle", []string{"needle", "haystack"}))
}

func TestInArrayWillReturnFalse(t *testing.T) {
	assert.False(t, InArray("needle", []string{"haystack"}))
}
