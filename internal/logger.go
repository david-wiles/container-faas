package internal

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type LogLevel int

const (
	logLevelInfo  LogLevel = 0
	logLevelWarn  LogLevel = 1
	logLevelError LogLevel = 2
	logLevelOff   LogLevel = 3
)

type Logger struct {
	infoLog  io.Writer
	errorLog io.Writer
	level    LogLevel
}

func (log Logger) LogRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.LogAccess(r)
			next.ServeHTTP(w, r)
		},
	)
}

func (log Logger) LogRequestFunc(next func(http.ResponseWriter, *http.Request)) http.Handler {
	return log.LogRequests(http.HandlerFunc(next))
}

func (log Logger) Error(message string) {
	if log.level < 3 {
		_, _ = fmt.Fprintln(log.errorLog, message)
	}
}

func (log Logger) Warning(message string) {
	if log.level < 2 {
		_, _ = fmt.Fprintln(log.infoLog, message)
	}
}

func (log Logger) Info(message string) {
	if log.level < 1 {
		_, _ = fmt.Fprintln(log.infoLog, time.Now().Format(time.RFC3339Nano), message)
	}
}

func (log Logger) LogAccess(r *http.Request) {
	if log.level < 1 {
		_, _ = fmt.Fprintln(log.infoLog, time.Now().Format(time.RFC3339Nano), r.Method, r.URL.Path)
	}
}

func (log Logger) LogError(err error) {
	if log.level < 3 {
		_, _ = fmt.Fprintln(log.errorLog, time.Now().Format(time.RFC3339Nano), err.Error())
	}
}
