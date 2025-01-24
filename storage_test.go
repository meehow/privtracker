package main

import (
	"net"
	"testing"
)

func BenchmarkPutPeerGetPeers(b *testing.B) {
	peer := NewPeer(net.ParseIP("127.0.0.1"), 6881)
	for i := 0; i < b.N; i++ {
		PutPeer("room", "infoHash", peer, true)
		GetPeers("room", "infoHash", peer, true, 99)
	}
}
