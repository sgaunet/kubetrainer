// Package views holds templ-generated HTML views and the Bootstrap asset handler.
//
//nolint:godoclint // templ-generated files attach version comments to the package declaration
package views

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

//go:generate go tool github.com/a-h/templ/cmd/templ generate

// BootStrapRootDir is the directory inside the embedded FS that contains Bootstrap assets.
const BootStrapRootDir = "bootstrap-5.2.3-dist"

// FsBootstrap embeds the Bootstrap distribution served by BootStrapHandler.
//
//go:embed bootstrap-5.2.3-dist/*
var FsBootstrap embed.FS

// BootStrapHandler returns an http.HandlerFunc that serves Bootstrap assets
// from the embedded filesystem, stripping subPathStripPrefix from request URLs.
func BootStrapHandler(subPathStripPrefix string) http.HandlerFunc {
	bootstrapFS, err := fs.Sub(FsBootstrap, BootStrapRootDir)
	if err != nil {
		panic(fmt.Errorf("failed getting the sub tree for the site files: %w", err))
	}
	handler := http.FileServer(http.FS(bootstrapFS))
	static := http.StripPrefix(subPathStripPrefix, handler)
	return func(w http.ResponseWriter, r *http.Request) {
		static.ServeHTTP(w, r)
	}
}
