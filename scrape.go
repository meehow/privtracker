package main

import (
	"crypto/sha1"
	"fmt"
	"net/http"

	"github.com/jackpal/bencode-go"
)

type ScrapeResponse struct {
	Files map[string]Stat `bencode:"files"`
}

type Stat struct {
	Complete   int `bencode:"complete"`
	Incomplete int `bencode:"incomplete"`
	// Downloaded uint `bencode:"downloaded"`
}

func scrape(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	infoHash := query.Get("info_hash")
	swarmHash := sha1.Sum([]byte(r.PathValue("room") + infoHash))

	numSeeders, numLeechers := GetStats(swarmHash)
	resp := ScrapeResponse{
		Files: map[string]Stat{
			infoHash: {
				Complete:   numSeeders,
				Incomplete: numLeechers,
			},
		},
	}
	w.Header().Add("X-PrivTracker", fmt.Sprintf("s:%d l:%d", numSeeders, numLeechers))
	if err := bencode.Marshal(w, resp); err != nil {
		http.Error(w, err.Error(), 400)
	}
}
