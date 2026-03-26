package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultVersion(t *testing.T) {
	assert.Equal(t, "2.0.x-dev", Version, "default version should be 2.0.x-dev")
}
