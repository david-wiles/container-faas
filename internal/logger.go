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

type loggedResponseWriter struct {
	http.ResponseWriter
	status   int
	headers  http.Header
	reqStart time.Time
}

func logResponseWriter(w http.ResponseWriter) *loggedResponseWriter {
	return &loggedResponseWriter{ResponseWriter: w}
}

func (l *loggedResponseWriter) Header() http.Header {
	return l.ResponseWriter.Header()
}

func (l *loggedResponseWriter) Write(b []byte) (int, error) {
	return l.ResponseWriter.Write(b)
}

func (l *loggedResponseWriter) WriteHeader(statusCode int) {
	l.ResponseWriter.WriteHeader(statusCode)
	l.status = statusCode
}

func (log Logger) LogRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if log.level > 0 {
				next.ServeHTTP(w, r)
			} else {
				loggedWriter := logResponseWriter(w)
				loggedWriter.reqStart = time.Now()
				next.ServeHTTP(loggedWriter, r)
				log.LogAccess(loggedWriter, r)
			}
		},
	)
}

func (log Logger) LogRequestFunc(next func(http.ResponseWriter, *http.Request)) http.Handler {
	return log.LogRequests(http.HandlerFunc(next))
}

func (log Logger) Error(message string) {
	if log.level < 3 {
		_, _ = fmt.Fprintf(log.errorLog, "[%s] ERROR: %s\n", timeNow(), message)
	}
}

func (log Logger) Warning(message string) {
	if log.level < 2 {
		_, _ = fmt.Fprintf(log.infoLog, "[%s] WARN: %s\n", timeNow(), message)
	}
}

func (log Logger) Info(message string) {
	if log.level < 1 {
		_, _ = fmt.Fprintf(log.infoLog, "[%s] INFO: %s\n", timeNow(), message)
	}
}

func (log Logger) LogAccess(w *loggedResponseWriter, r *http.Request) {
	if log.level < 1 {
		remoteAddr := r.Header.Get("X-Forwarded-For")
		userAgent := r.Header.Get("User-Agent")
		timing := time.Now().Sub(w.reqStart).Milliseconds()
		// $remote_addr [$time_local] "$request" $path $status $http_user_agent $request_time
		_, _ = fmt.Fprintf(log.infoLog, "[%s] %s %q %q %d %s %dms\n", timeNow(), remoteAddr, r.Method, r.URL.Path, w.status, userAgent, timing)
	}
}

func (log Logger) LogError(err error) {
	log.Error(err.Error())
}

func timeNow() string {
	return time.Now().Format(time.RFC3339Nano)
}
