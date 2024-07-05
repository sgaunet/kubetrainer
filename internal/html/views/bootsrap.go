package views

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

//go:generate templ generate

const BootStrapRootDir = "bootstrap-5.2.3-dist"

//go:embed bootstrap-5.2.3-dist/*
var FsBootstrap embed.FS

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
