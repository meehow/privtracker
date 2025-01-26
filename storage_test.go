package main

import (
	"crypto/rand"
	mrand "math/rand"
	"net"
	"testing"
)

func BenchmarkPutPeerGetPeers(b *testing.B) {
	var swarmHash Hash
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			rand.Read(swarmHash[:])
			for k := 0; k < 10; k++ {
				peer := NewPeer(net.ParseIP("127.0.0.1"), uint16(mrand.Uint32()))

				PutPeer(swarmHash, peer, true)
				GetPeers(swarmHash, peer, true, 99)

				GraduateLeecher(swarmHash, peer)
				GetPeers(swarmHash, peer, true, 99)

				DeletePeer(swarmHash, peer)
				GetPeers(swarmHash, peer, true, 99)
			}
		}
	}
}
