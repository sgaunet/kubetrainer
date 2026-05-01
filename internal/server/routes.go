// Package server exposes the kubetrainer HTTP web server and route definitions.
package server

import (
	"github.com/sgaunet/kubetrainer/internal/html/views"
)

// PublicRoutes registers the public HTTP routes on the underlying router.
func (w *WebServer) PublicRoutes() {
	w.router.Get("/", w.htmlController.Index)
	w.router.Handle("/bootstrap-5.2.3-dist/*", views.BootStrapHandler("/bootstrap-5.2.3-dist/"))
	w.router.Get("/readiness", w.htmlController.Readiness)
	w.router.Get("/liveness", w.htmlController.Liveness)
	w.router.Get("/update-liveness", w.htmlController.ChangeLivenessState)
	w.router.Get("/update-readiness", w.htmlController.ChangeReadinessState)
	w.router.Post("/publish-time", w.htmlController.PublishTime)
	w.router.Post("/publish-time/{count}", w.htmlController.PublishTime)
}
