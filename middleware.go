package main

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"
)

type Middleware func(http.Handler) http.Handler

type ResponseWriterWrapper struct {
	http.ResponseWriter
	StatusCode int
	BytesSent  int
}

func (rw *ResponseWriterWrapper) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *ResponseWriterWrapper) Write(data []byte) (int, error) {
	bytesWritten, err := rw.ResponseWriter.Write(data)
	rw.BytesSent += bytesWritten
	return bytesWritten, err
}

func logRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapper := &ResponseWriterWrapper{ResponseWriter: w, StatusCode: http.StatusOK}
		timer := time.Now()
		next.ServeHTTP(wrapper, r)
		fmt.Printf("%d - %5dµs %47s %4s %42s %6d %11s - %s - %s\n",
			wrapper.StatusCode, time.Since(timer).Microseconds(), r.RemoteAddr,
			r.Method, r.URL.Path, wrapper.BytesSent,
			wrapper.Header().Get("X-PrivTracker"), r.Referer(), r.UserAgent())
	})
}

func headersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000") // hsts
		w.Header().Set("Server", "PrivTracker")
		next.ServeHTTP(w, r)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("Recovered from panic: %v\n%s\n", err, debug.Stack())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func chainMiddleware(handler http.Handler, middlewares ...Middleware) http.Handler {
	// Apply middlewares in reverse order (outermost to innermost)
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
