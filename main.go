package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jackpal/bencode-go"
)

func main() {
	r := gin.Default()
	r.GET("/:room/announce", announce)
	r.GET("/:room/scrape", scrape)
	go Cleanup()
	r.Run()
}

type AnnounceRequest struct {
	InfoHash      string `form:"info_hash"`
	PeerID        string `form:"peer_id"`
	IP            string `form:"ip"`
	Port          uint16 `form:"port"`
	Uploaded      uint   `form:"uploaded"`
	Downloaded    uint   `form:"downloaded"`
	Left          uint   `form:"left"`
	Numwant       uint   `form:"numwant"`
	Key           string `form:"key"`
	Compact       bool   `form:"compact"`
	SupportCrypto bool   `form:"supportcrypto"`
	Event         string `form:"event"`
}

func (req *AnnounceRequest) IsSeeding() bool {
	return req.Left == 0
}

type AnnounceResponse struct {
	Interval   int    `bencode:"interval"`
	Complete   int    `bencode:"complete"`
	Incomplete int    `bencode:"incomplete"`
	Peers      string `bencode:"peers"`
	PeersIPv6  string `bencode:"peers_ipv6"`
}

func announce(c *gin.Context) {
	req := new(AnnounceRequest)
	c.BindQuery(req)
	// if req.IP == "" {
	req.IP = c.ClientIP() // not sure if ip from request should be honored
	// }
	if req.Numwant == 0 {
		req.Numwant = 30
	}
	switch req.Event {
	case "stopped":
		DeletePeer(c.Param("room"), req.InfoHash, req.IP, req.Port)
	case "completed":
		GraduateLeecher(c.Param("room"), req.InfoHash, req.IP, req.Port)
	default:
		PutPeer(c.Param("room"), req.InfoHash, req.IP, req.Port, req.IsSeeding())
	}
	peersv4, peersv6, numSeeders, numLeechers := GetPeers(c.Param("room"), req.InfoHash, req.Numwant, req.IsSeeding())
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
		Peers:      string(peersv4),
		PeersIPv6:  string(peersv6),
	}
	if err := bencode.Marshal(c.Writer, resp); err != nil {
		c.Error(err)
		return
	}
}

type ScrapeRequest struct {
	InfoHash string `form:"info_hash"`
}

type ScrapeResponse struct {
	Files map[string]Stat `bencode:"files"`
}

type Stat struct {
	Complete   int `bencode:"complete"`
	Incomplete int `bencode:"incomplete"`
	// Downloaded uint `bencode:"downloaded"`
}

func scrape(c *gin.Context) {
	req := new(ScrapeRequest)
	c.BindQuery(req)
	numSeeders, numLeechers := GetStats(c.Param("room"), req.InfoHash)
	resp := ScrapeResponse{
		Files: map[string]Stat{
			req.InfoHash: {
				Complete:   numSeeders,
				Incomplete: numLeechers,
			},
		},
	}
	if err := bencode.Marshal(c.Writer, resp); err != nil {
		c.Error(err)
		return
	}
}
