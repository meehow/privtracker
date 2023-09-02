package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"golang.org/x/crypto/acme/autocert"
)

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
		config.TrustedProxies = []string{"127.0.0.1"}
		config.ProxyHeader = fiber.HeaderXForwardedFor
	}

	app := fiber.New(config)
	app.Use(recover.New())
	// app.Use(pprof.New())
	app.Use(myLogger())
	app.Use(hsts)
	app.Get("/", docs)
	app.Static("/", "docs", fiber.Static{
		MaxAge:        3600 * 24 * 7,
		Compress:      true,
		CacheDuration: time.Hour,
	})
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
	domain string
	port   string
}

func getEnvConfig() EnvConfig {

	config := EnvConfig{
		domain: "privtracker.com",
		port:   "1337",
	}

	port := os.Getenv("PORT")
	domain := os.Getenv("DOMAIN")
	if domain != "" {
		config.domain = domain
	}
	if port != "" {
		config.port = port
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
