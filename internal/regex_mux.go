package internal

import (
	"net/http"
	"regexp"
)

type RegexMux struct {
	routes   []*route
	NotFound http.Handler
}

type route struct {
	pattern *regexp.Regexp
	handler http.Handler
}

type httpHandler struct {
	handler func(w http.ResponseWriter, r *http.Request)
}

func (h httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler(w, r)
}

func (mux *RegexMux) Handle(pattern string, handler http.Handler) {
	re := regexp.MustCompile(pattern)
	mux.routes = append(mux.routes, &route{re, handler})
}

func (mux *RegexMux) HandleFunc(pattern string, handler func(w http.ResponseWriter, r *http.Request)) {
	re := regexp.MustCompile(pattern)
	mux.routes = append(mux.routes, &route{re, httpHandler{handler}})
}

func (mux *RegexMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Call the handler which matches the pattern
	for _, route := range mux.routes {
		if route.pattern.MatchString(r.URL.Path) {
			route.handler.ServeHTTP(w, r)
			return
		}
	}
	// Call the 404 handler if no patterns are matched
	mux.NotFound.ServeHTTP(w, r)
}
