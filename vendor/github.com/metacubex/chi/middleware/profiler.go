//go:build !tinygo
// +build !tinygo

package middleware

import (
	"expvar"
	"fmt"

	"github.com/metacubex/http"
	"github.com/metacubex/http/pprof"

	"github.com/metacubex/chi"
)

// Profiler is a convenient subrouter used for mounting net/http/pprof. ie.
//
//	func MyService() http.Handler {
//		r := chi.NewRouter()
//		// ..middlewares
//		r.Mount("/debug", middleware.Profiler())
//		// ..routes
//		return r
//	}
func Profiler() http.Handler {
	r := chi.NewRouter()
	r.Use(NoCache)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.RequestURI+"/pprof/", http.StatusMovedPermanently)
	})
	r.HandleFunc("/pprof", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.RequestURI+"/", http.StatusMovedPermanently)
	})

	r.HandleFunc("/pprof/*", pprof.Index)
	r.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/pprof/profile", pprof.Profile)
	r.HandleFunc("/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/pprof/trace", pprof.Trace)
	r.Handle("/vars", http.HandlerFunc(expvarHandler))

	r.Handle("/pprof/goroutine", pprof.Handler("goroutine"))
	r.Handle("/pprof/threadcreate", pprof.Handler("threadcreate"))
	r.Handle("/pprof/mutex", pprof.Handler("mutex"))
	r.Handle("/pprof/heap", pprof.Handler("heap"))
	r.Handle("/pprof/block", pprof.Handler("block"))
	r.Handle("/pprof/allocs", pprof.Handler("allocs"))

	return r
}

func expvarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}
