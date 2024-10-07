package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMd5WithCorrectParameterWillReturnCorrectHash(t *testing.T) {
	assert.Equal(t, "098f6bcd4621d373cade4e832627b4f6", Md5([]byte("test")))
}

func TestMd5WithEmptyParameterWillReturnEmptyHash(t *testing.T) {
	assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", Md5([]byte{}))
}
