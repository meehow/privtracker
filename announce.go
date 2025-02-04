package main

import (
	"crypto/sha1"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackpal/bencode-go"
)

type AnnounceResponse struct {
	Interval   int    `bencode:"interval"`
	Complete   int    `bencode:"complete"`
	Incomplete int    `bencode:"incomplete"`
	Peers      []byte `bencode:"peers"`
	PeersIPv6  []byte `bencode:"peers_ipv6"`
}

func announce(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	port, err := strconv.ParseUint(query.Get("port"), 10, 16)
	if err != nil {
		http.Error(w, "port missing", 400)
		return
	}
	ip := getRemoteIP(r)
	if ip == nil {
		http.Error(w, "can't parse IP", 400)
		return
	}
	numwant, err := strconv.Atoi(query.Get("numwant"))
	if err != nil || numwant < 1 {
		numwant = 30
	}
	swarmHash := sha1.Sum([]byte(r.PathValue("room") + query.Get("info_hash")))
	peer := NewPeer(ip, uint16(port))
	isSeeding := query.Get("left") == "0"
	switch query.Get("event") {
	case "stopped":
		DeletePeer(swarmHash, peer)
	case "completed":
		GraduateLeecher(swarmHash, peer)
	default:
		PutPeer(swarmHash, peer, isSeeding)
	}
	peersIPv4, peersIPv6, numSeeders, numLeechers := GetPeers(swarmHash, peer, isSeeding, numwant)
	interval := 480 // must be smaller than cleanup interval
	switch {
	case numSeeders+numLeechers < 10:
		// try to synchronize peer requests. Maybe it will help µTP with UDP port punching... not sure
		interval = 240 - int(time.Now().Unix()+int64(swarmHash[0]))%240
		if interval < 60 {
			interval += 240
		}
	case numSeeders+numLeechers > 30:
		interval = 900
	}
	resp := AnnounceResponse{
		Interval:   interval,
		Complete:   numSeeders,
		Incomplete: numLeechers,
		Peers:      peersIPv4,
		PeersIPv6:  peersIPv6,
	}
	w.Header().Add("X-PrivTracker", fmt.Sprintf("s:%d l:%d", numSeeders, numLeechers))
	if err := bencode.Marshal(w, resp); err != nil {
		http.Error(w, err.Error(), 400)
	}
}

func getRemoteIP(r *http.Request) net.IP {
	addr := r.RemoteAddr
	if colonIndex := strings.LastIndex(addr, ":"); colonIndex != -1 {
		addr = addr[:colonIndex]
	}
	addr = strings.Trim(addr, "[]")
	ip := net.ParseIP(addr)
	if ip.IsPrivate() {
		ips := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
		if len(ips) > 0 {
			ipForwarded := net.ParseIP(strings.TrimSpace(ips[0]))
			if ipForwarded != nil {
				ip = ipForwarded
			}
		}
	}
	return ip
}
