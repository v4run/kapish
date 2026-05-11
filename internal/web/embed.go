package web

import (
	"embed"
	"io/fs"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

// frontendRoot returns an fs.FS rooted at frontend/dist.
func frontendRoot() fs.FS {
	sub, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		panic(err) // build-time guarantee; can't happen if //go:embed succeeded
	}
	return sub
}
