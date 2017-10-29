package debug

import (
	"expvar"
	"net/http"
	"net/http/pprof"
)

func NewServer(addr string) *http.Server {
	pprofHandler := http.NewServeMux()
	pprofHandler.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	pprofHandler.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	pprofHandler.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	pprofHandler.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	pprofHandler.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	expvarHandler := expvar.Handler()
	pprofHandler.Handle("/debug/vars", expvarHandler)
	pprofHandler.Handle("/debug/pprof/vars", expvarHandler) // alias

	return &http.Server{Addr: addr, Handler: pprofHandler}
}
