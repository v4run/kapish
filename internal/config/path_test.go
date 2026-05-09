package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolvePath_FlagWins(t *testing.T) {
	p, err := ResolvePath(PathSources{
		Flag:          "/tmp/explicit.yaml",
		EnvVar:        "/tmp/env.yaml",
		XDGConfigHome: "/tmp/xdg",
	})
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/explicit.yaml", p)
}

func TestResolvePath_EnvWhenNoFlag(t *testing.T) {
	p, err := ResolvePath(PathSources{
		EnvVar:        "/tmp/env.yaml",
		XDGConfigHome: "/tmp/xdg",
	})
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/env.yaml", p)
}

func TestResolvePath_XDGWhenNoFlagNoEnv(t *testing.T) {
	p, err := ResolvePath(PathSources{
		XDGConfigHome: "/tmp/xdg",
	})
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join("/tmp/xdg", "kapish", "config.yaml"), p)
}

func TestResolvePath_HomeFallback(t *testing.T) {
	p, err := ResolvePath(PathSources{
		Home: "/users/foo",
	})
	assert.NoError(t, err)
	assert.Equal(t, "/users/foo/.config/kapish/config.yaml", p)
}

func TestResolvePath_NoSources(t *testing.T) {
	_, err := ResolvePath(PathSources{})
	assert.Error(t, err)
}
