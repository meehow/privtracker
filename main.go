package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"golang.org/x/crypto/acme/autocert"
)

var port = os.Getenv("PORT")

func main() {
	tlsEnabled := port == "443"
	if port == "" {
		port = "1337"
	}
	config := fiber.Config{
		ServerHeader: "privtracker.com",
		ReadTimeout:  time.Second * 245,
		WriteTimeout: time.Second * 15,
		Network:      fiber.NetworkTCP,
	}
	// if you disable TLS, then I guess you want to use existing proxy
	if !tlsEnabled {
		config.EnableTrustedProxyCheck = true
		config.TrustedProxies = []string{"127.0.0.1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
		config.ProxyHeader = fiber.HeaderXForwardedFor
	}

	go Cleanup()

	app := fiber.New(config)
	app.Use(recover.New())
	// app.Use(pprof.New())
	app.Use(myLogger())
	app.Use(hsts)
	app.Static("/", "docs", fiber.Static{
		MaxAge:        3600 * 24 * 7,
		Compress:      true,
		CacheDuration: time.Hour,
	})
	app.Get("/dashboard", monitor.New())
	app.Get("/:room/announce", announce)
	app.Get("/:room/scrape", scrape)
	app.Server().LogAllErrors = true
	if tlsEnabled {
		go redirect80(config)
		log.Fatal(app.Listener(autocertListener()))
	} else {
		log.Fatal(app.Listen(":" + port))
	}
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
		NextProtos: []string{
			"http/1.1", "acme-tls/1",
		},
	}
	ln, err := tls.Listen("tcp", ":443", cfg)
	if err != nil {
		log.Fatal(err)
	}
	return ln
}

func redirect80(config fiber.Config) {
	config.DisableStartupMessage = true
	app := fiber.New(config)
	app.Use(func(c *fiber.Ctx) error {
		return c.Redirect(fmt.Sprintf("https://%s/", c.Hostname()), fiber.StatusMovedPermanently)
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
