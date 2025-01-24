package main

import (
	"net"

	"github.com/gofiber/fiber/v2"
	"github.com/jackpal/bencode-go"
)

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
	room := c.Params("room")
	ip := net.ParseIP(c.IP())
	if ip == nil {
		ip = c.Context().RemoteIP()
	}
	if req.Numwant == 0 {
		req.Numwant = 30
	}
	peer := NewPeer(ip, req.Port)
	switch req.Event {
	case "stopped":
		DeletePeer(room, req.InfoHash, peer)
	case "completed":
		GraduateLeecher(room, req.InfoHash, peer)
	default:
		PutPeer(room, req.InfoHash, peer, req.IsSeeding())
	}
	peersIPv4, peersIPv6, numSeeders, numLeechers := GetPeers(room, req.InfoHash, peer, req.IsSeeding(), req.Numwant)
	interval := 120
	if numSeeders == 0 {
		interval /= 2
	} else if numLeechers == 0 {
		interval *= 4
	}
	resp := AnnounceResponse{
		Interval:   interval,
		Complete:   numSeeders,
		Incomplete: numLeechers,
		Peers:      peersIPv4,
		PeersIPv6:  peersIPv6,
	}
	return bencode.Marshal(c, resp)
}
