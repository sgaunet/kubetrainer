package server

import (
	"github.com/sgaunet/kubetrainer/internal/html/views"
)

func (w *WebServer) PublicRoutes() {
	w.router.Get("/", w.htmlController.Index)
	w.router.Handle("/bootstrap-5.2.3-dist/*", views.BootStrapHandler("/bootstrap-5.2.3-dist/"))
	// w.router.Get("/static/logo.svg", w.htmlController.Logo)
	w.router.Get("/readiness", w.htmlController.Readiness)
	w.router.Get("/liveness", w.htmlController.Liveness)
	w.router.Get("/update-liveness", w.htmlController.ChangeLivenessState)
	w.router.Get("/update-readiness", w.htmlController.ChangeReadinessState)
	w.router.Post("/publish-time", w.htmlController.PublishTime)
}

// PrintMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
// func PrintMemUsage() {
// 	var m runtime.MemStats
// 	runtime.ReadMemStats(&m)
// 	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
// 	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
// 	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
// 	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
// 	fmt.Printf("\tNumGC = %v\n", m.NumGC)
// }

// func bToMb(b uint64) uint64 {
// return b / 1024 / 1024
// }
