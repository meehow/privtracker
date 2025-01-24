package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

type Hash [20]byte // we use sha1 and we are not affraid of hash collisions
type Peer [18]byte // 16 bytes for IP and 2 bytes for port number

func NewPeer(ip net.IP, port uint16) (peer Peer) {
	copy(peer[:], ip)
	peer[16] = byte(port >> 8)
	peer[17] = byte(port)
	return
}

func (peer Peer) String() string {
	return fmt.Sprintf("%s:%d", net.IP(peer[:16]), binary.BigEndian.Uint16(peer[16:]))
}

type shard struct {
	swarms map[Hash]Swarm
	sync.RWMutex
}

type Swarm struct {
	seeders  map[Peer]int64
	leechers map[Peer]int64
}

func NewSwarm() Swarm {
	return Swarm{
		seeders:  make(map[Peer]int64),
		leechers: make(map[Peer]int64),
	}
}

var shards = NewShards(512)
var v4InV6Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}

func shardIndex(hash [20]byte) int {
	return int(binary.BigEndian.Uint16(hash[:2])) % len(shards)
}

func NewShards(size int) []*shard {
	shards := make([]*shard, size)
	for i := 0; i < size; i++ {
		shards[i] = &shard{
			swarms: make(map[Hash]Swarm),
		}
	}
	return shards
}

func PutPeer(room, infoHash string, peer Peer, seeding bool) {
	h := sha1.Sum([]byte(room + infoHash))
	shard := shards[shardIndex(h)]
	shard.Lock()
	if _, ok := shard.swarms[h]; !ok {
		shard.swarms[h] = NewSwarm()
	}
	if seeding {
		shard.swarms[h].seeders[peer] = time.Now().Unix()
	} else {
		shard.swarms[h].leechers[peer] = time.Now().Unix()
	}
	shard.Unlock()
}

func DeletePeer(room, infoHash string, peer Peer) {
	h := sha1.Sum([]byte(room + infoHash))
	shard := shards[shardIndex(h)]
	shard.Lock()
	if _, ok := shard.swarms[h]; !ok {
		return
	}
	delete(shard.swarms[h].seeders, peer)
	delete(shard.swarms[h].leechers, peer)
	shard.Unlock()
}

func GraduateLeecher(room, infoHash string, peer Peer) {
	h := sha1.Sum([]byte(room + infoHash))
	shard := shards[shardIndex(h)]
	shard.Lock()
	if _, ok := shard.swarms[h]; !ok {
		shard.swarms[h] = NewSwarm()
	}
	shard.swarms[h].seeders[peer] = time.Now().Unix()
	delete(shard.swarms[h].leechers, peer)
	shard.Unlock()
}

func GetPeers(room, infoHash string, client Peer, seeding bool, numWant uint) (peersIPv4, peersIPv6 []byte, numSeeders, numLeechers int) {
	h := sha1.Sum([]byte(room + infoHash))
	shard := shards[shardIndex(h)]
	shard.RLock()
	// seeders don't need other seeders
	if !seeding {
		for peer := range shard.swarms[h].seeders {
			if numWant == 0 {
				break
			}
			if bytes.HasPrefix(peer[:], v4InV6Prefix) {
				peersIPv4 = append(peersIPv4, peer[len(v4InV6Prefix):]...)
			} else {
				peersIPv6 = append(peersIPv6, peer[:]...)
			}
			numWant--
		}
	}
	for peer := range shard.swarms[h].leechers {
		if peer == client {
			continue
		}
		if numWant == 0 {
			break
		}
		if bytes.HasPrefix(peer[:], v4InV6Prefix) {
			peersIPv4 = append(peersIPv4, peer[12:]...)
		} else {
			peersIPv6 = append(peersIPv6, peer[:]...)
		}
		numWant--
	}
	numSeeders = len(shard.swarms[h].seeders)
	numLeechers = len(shard.swarms[h].leechers)
	shard.RUnlock()
	return
}

func GetStats(room, infoHash string) (numSeeders, numLeechers int) {
	h := sha1.Sum([]byte(room + infoHash))
	shard := shards[shardIndex(h)]
	shard.RLock()
	numSeeders = len(shard.swarms[h].seeders)
	numLeechers = len(shard.swarms[h].leechers)
	shard.RUnlock()
	return
}

func Cleanup(duration time.Duration) {
	ticker := time.NewTicker(duration)
	for range ticker.C {
		expiration := time.Now().Unix() - int64(duration.Seconds())
		for _, shard := range shards {
			shard.Lock()
			for h, swarm := range shard.swarms {
				for peer, lastSeen := range swarm.seeders {
					if lastSeen < expiration {
						delete(swarm.seeders, peer)
					}
				}
				for peer, lastSeen := range swarm.leechers {
					if lastSeen < expiration {
						delete(swarm.leechers, peer)
					}
				}
				if len(swarm.leechers) == 0 && len(swarm.seeders) == 0 {
					delete(shard.swarms, h)
				}
			}
			shard.Unlock()
		}
	}
}
