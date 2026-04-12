package main

import (
	"testing"

	themeview "github.com/buzyka/imlate/internal/usecase/theme"
	"github.com/golobby/container/v3"
	"github.com/stretchr/testify/assert"
)

func TestResolveThemeService_ReturnsContainerSingleton(t *testing.T) {
	c := container.New()
	singleton := &themeview.Service{}
	container.MustSingleton(c, func() *themeview.Service {
		return singleton
	})

	resolved := resolveThemeService(c)

	assert.Same(t, singleton, resolved)
}
