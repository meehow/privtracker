package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/crypto/acme/autocert"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "1337"
	}
	handler := router(recoveryMiddleware, headersMiddleware, logRequestMiddleware)
	if port == "443" {
		go redirect80()
		fmt.Println("PrivTracker listening on https://0.0.0.0/")
		log.Fatal(http.Serve(autocertListener(), handler))
	} else {
		fmt.Printf("PrivTracker listening on http://0.0.0.0:%s/\n", port)
		log.Fatal(http.ListenAndServe(":"+port, handler))
	}
}

func router(middlewares ...Middleware) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("docs")))
	mux.HandleFunc("GET /{room}/announce", announce)
	mux.HandleFunc("GET /{room}/scrape", scrape)
	return chainMiddleware(mux, middlewares...)
}

func autocertListener() net.Listener {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	cacheDir := filepath.Join(homeDir, ".cache", "golang-autocert")
	m := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(cacheDir),
	}
	cfg := &tls.Config{
		GetCertificate: m.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1", "acme-tls/1"},
	}
	listener, err := tls.Listen("tcp", ":443", cfg)
	if err != nil {
		log.Fatal(err)
	}
	return listener
}

func redirect(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("https://%s/", r.Host)
	http.Redirect(w, r, url, http.StatusMovedPermanently)
}

func redirect80() {
	handler := chainMiddleware(http.HandlerFunc(redirect), logRequestMiddleware)
	err := http.ListenAndServe(":80", handler)
	if err != nil {
		fmt.Println(err)
	}
}
