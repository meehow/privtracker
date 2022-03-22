package main

import (
	"crypto/tls"
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
	"github.com/jackpal/bencode-go"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	go Cleanup()
	config := fiber.Config{
		ServerHeader: "privtracker",
		ReadTimeout:  time.Second * 125,
	}
	domains, tls := os.LookupEnv("DOMAINS")
	if !tls {
		config.EnableTrustedProxyCheck = true
		config.TrustedProxies = []string{"127.0.0.1"}
		config.ProxyHeader = fiber.HeaderXForwardedFor
	}
	app := fiber.New(config)
	app.Use(recover.New())
	app.Use(myLogger())
	app.Use(hsts)
	app.Get("/", docs)
	app.Static("/", "docs", fiber.Static{MaxAge: 3600 * 24 * 7})
	app.Get("/dashboard", monitor.New())
	app.Get("/:room/announce", announce)
	app.Get("/:room/scrape", scrape)
	app.Server().LogAllErrors = true
	if tls {
		go redirect80(config)
		split := strings.Split(domains, ",")
		log.Fatal(app.Listener(myListener(split...)))
	} else {
		port := os.Getenv("PORT")
		if port == "" {
			port = "1337"
		}
		log.Fatal(app.Listen(":" + port))
	}
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
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
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
		return c.Redirect("https://privtracker.com/", fiber.StatusMovedPermanently)
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
	if c.Hostname() != "privtracker.com" {
		return c.Redirect("https://privtracker.com/", fiber.StatusMovedPermanently)
	}
	return c.Next()
}

type AnnounceRequest struct {
	InfoHash      string `query:"info_hash"`
	PeerID        string `query:"peer_id"`
	IP            string `query:"ip"`
	Port          uint16 `query:"port"`
	Uploaded      uint   `query:"uploaded"`
	Downloaded    uint   `query:"downloaded"`
	Left          uint   `query:"left"`
	Numwant       uint   `query:"numwant"`
	Key           string `query:"key"`
	Compact       bool   `query:"compact"`
	SupportCrypto bool   `query:"supportcrypto"`
	Event         string `query:"event"`
}

func (req *AnnounceRequest) IsSeeding() bool {
	return req.Left == 0
}

type AnnounceResponse struct {
	Interval   int    `bencode:"interval"`
	Complete   int    `bencode:"complete"`
	Incomplete int    `bencode:"incomplete"`
	Peers      []byte `bencode:"peers"`
	PeersIPv6  []byte `bencode:"peers_ipv6"`
}

func announce(c *fiber.Ctx) error {
	var req AnnounceRequest
	err := c.QueryParser(&req)
	if err != nil {
		return err
	}
	req.IP = c.IP()
	if req.Numwant == 0 {
		req.Numwant = 30
	}
	switch req.Event {
	case "stopped":
		DeletePeer(c.Params("room"), req.InfoHash, req.IP, req.Port)
	case "completed":
		GraduateLeecher(c.Params("room"), req.InfoHash, req.IP, req.Port)
	default:
		PutPeer(c.Params("room"), req.InfoHash, req.IP, req.Port, req.IsSeeding())
	}
	peersIPv4, peersIPv6, numSeeders, numLeechers := GetPeers(c.Params("room"), req.InfoHash, req.IP, req.Port, req.IsSeeding(), req.Numwant)
	interval := 120
	if numSeeders == 0 {
		interval /= 2
	} else if numLeechers == 0 {
		interval *= 2
	}
	resp := AnnounceResponse{
		Interval:   interval,
		Complete:   numSeeders,
		Incomplete: numLeechers,
		Peers:      peersIPv4,
		PeersIPv6:  peersIPv6,
	}
	defer c.Response().SetConnectionClose()
	return bencode.Marshal(c, resp)
}

type ScrapeRequest struct {
	InfoHash string `query:"info_hash"`
}

type ScrapeResponse struct {
	Files map[string]Stat `bencode:"files"`
}

type Stat struct {
	Complete   int `bencode:"complete"`
	Incomplete int `bencode:"incomplete"`
	// Downloaded uint `bencode:"downloaded"`
}

func scrape(c *fiber.Ctx) error {
	var req ScrapeRequest
	err := c.QueryParser(&req)
	if err != nil {
		return err
	}
	numSeeders, numLeechers := GetStats(c.Params("room"), req.InfoHash)
	resp := ScrapeResponse{
		Files: map[string]Stat{
			req.InfoHash: {
				Complete:   numSeeders,
				Incomplete: numLeechers,
			},
		},
	}
	return bencode.Marshal(c, resp)
}
