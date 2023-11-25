package main

import (
	"crypto/tls"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"golang.org/x/crypto/acme/autocert"
)

//go:embed docs
var embedDirStatic embed.FS
var envConfig = getEnvConfig()

func main() {
	go Cleanup()
	config := fiber.Config{
		ServerHeader: "privtracker",
		ReadTimeout:  time.Second * 245,
		WriteTimeout: time.Second * 15,
		Network:      fiber.NetworkTCP,
	}
	domains, tls := os.LookupEnv("DOMAINS")
	if !tls {
		config.EnableTrustedProxyCheck = true
		config.TrustedProxies = envConfig.trustedProxies
		config.ProxyHeader = fiber.HeaderXForwardedFor
	}

	docsRoot, err := fs.Sub(embedDirStatic, "docs")
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New(config)
	app.Use(recover.New())
	// app.Use(pprof.New())
	app.Use(myLogger())
	app.Use(hsts)
	app.Get("/", docs)
	app.Use("/", filesystem.New(filesystem.Config{
		Root:   http.FS(docsRoot),
		MaxAge: 3600 * 24 * 7,
	}))
	app.Get("/dashboard", monitor.New())
	app.Get("/:room/announce", announce)
	app.Get("/:room/scrape", scrape)
	app.Server().LogAllErrors = true
	if tls {
		go redirect80(config)
		split := strings.Split(domains, ",")
		log.Fatal(app.Listener(myListener(split...)))
	} else {
		log.Fatal(app.Listen(":" + envConfig.port))
	}
}

type EnvConfig struct {
	domain         string
	port           string
	trustedProxies []string
}

func getEnvConfig() EnvConfig {

	config := EnvConfig{
		domain:         "privtracker.com",
		port:           "1337",
		trustedProxies: []string{"127.0.0.1"},
	}

	port := os.Getenv("PORT")
	domain := os.Getenv("DOMAIN")
	trustedProxies := os.Getenv("TRUSTED_PROXIES")
	if domain != "" {
		config.domain = domain
	}
	if port != "" {
		config.port = port
	}
	if trustedProxies != "" {
		config.trustedProxies = strings.Split(trustedProxies, ",")
	}

	return config
}

func myListener(domains ...string) net.Listener {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	cacheDir := filepath.Join(homeDir, ".cache", "golang-autocert")
	m := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domains...),
		Cache:      autocert.DirCache(cacheDir),
	}
	cfg := &tls.Config{
		GetCertificate: m.GetCertificate,
		NextProtos: []string{
			"http/1.1", "acme-tls/1",
		},
		// MinVersion: tls.VersionTLS12,
		// CipherSuites: []uint16{
		// 	tls.TLS_AES_128_GCM_SHA256,
		// 	tls.TLS_AES_256_GCM_SHA384,
		// 	tls.TLS_CHACHA20_POLY1305_SHA256,
		// 	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		// 	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		// 	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		// 	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		// 	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		// 	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		// },
	}
	ln, err := tls.Listen("tcp", ":443", cfg)
	if err != nil {
		panic(err)
	}
	return ln
}

func redirect80(config fiber.Config) {
	config.DisableStartupMessage = true
	app := fiber.New(config)
	app.Use(func(c *fiber.Ctx) error {
		return c.Redirect(fmt.Sprintf("https://%s/", envConfig.domain), fiber.StatusMovedPermanently)
	})
	log.Print(app.Listen(":80"))
}

func myLogger() fiber.Handler {
	loggerConfig := logger.ConfigDefault
	loggerConfig.Format = "${status} - ${latency} ${ip} ${method} ${path} ${bytesSent} - ${referer} - ${ua}\n"
	return logger.New(loggerConfig)
}

func hsts(c *fiber.Ctx) error {
	c.Set("Strict-Transport-Security", "max-age=31536000")
	return c.Next()
}

func docs(c *fiber.Ctx) error {
	if c.Hostname() != envConfig.domain {
		return c.Redirect(fmt.Sprintf("https://%s/", envConfig.domain), fiber.StatusMovedPermanently)
	}
	return c.Next()
}
