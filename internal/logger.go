package internal

import (
	"fmt"
	"net/http"
	"time"
)

type Logger struct {
	accessLog  string
	errorLog   string
	errorLevel int
}

func (log *Logger) LogRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.LogAccess(r)
			next.ServeHTTP(w, r)
			log.Info("Responded to request")
		},
	)
}

func (log *Logger) LogRequestFunc(next func(http.ResponseWriter, *http.Request)) http.Handler {
	return log.LogRequests(http.HandlerFunc(next))
}

func (log *Logger) Error(message string) {

}

func (log *Logger) Warning(message string) {

}

func (log *Logger) Info(message string) {
	_, _ = fmt.Println(time.Now().Format(time.RFC3339Nano), message)
}

func (log *Logger) LogAccess(r *http.Request) {
	_, _ = fmt.Println(time.Now().Format(time.RFC3339Nano), r.Method, r.URL.Path)
}

func (log *Logger) LogError(err error) {

}
