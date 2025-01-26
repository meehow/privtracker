package main

import (
	"crypto/sha1"

	"github.com/gofiber/fiber/v2"
	"github.com/jackpal/bencode-go"
)

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
	swarmHash := sha1.Sum([]byte(c.Params("room") + req.InfoHash))
	numSeeders, numLeechers := GetStats(swarmHash)
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
