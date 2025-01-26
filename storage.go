package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"
	"time"
)

type Hash [sha1.Size]byte // we use sha1 and we are not affraid of hash collisions
type Peer [18]byte        // 16 bytes for IP and 2 bytes for port number

var shards = NewShards(256)
var v4InV6Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}

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

func NewPeer(ip net.IP, port uint16) (peer Peer) {
	copy(peer[:16], ip)
	binary.BigEndian.PutUint16(peer[16:18], port)
	return
}

func (peer Peer) String() string {
	ip := net.IP(peer[:16])
	port := binary.BigEndian.Uint16(peer[16:18])
	return fmt.Sprintf("%s:%d", ip, port)
}

func shardIndex(hash Hash) int {
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

func PutPeer(h Hash, peer Peer, seeding bool) {
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

func DeletePeer(h Hash, peer Peer) {
	shard := shards[shardIndex(h)]
	shard.Lock()
	if _, ok := shard.swarms[h]; !ok {
		return
	}
	delete(shard.swarms[h].seeders, peer)
	delete(shard.swarms[h].leechers, peer)
	shard.Unlock()
}

func GraduateLeecher(h Hash, peer Peer) {
	shard := shards[shardIndex(h)]
	shard.Lock()
	if _, ok := shard.swarms[h]; !ok {
		shard.swarms[h] = NewSwarm()
	}
	shard.swarms[h].seeders[peer] = time.Now().Unix()
	delete(shard.swarms[h].leechers, peer)
	shard.Unlock()
}

func GetPeers(h Hash, client Peer, seeding bool, numWant uint) (peersIPv4, peersIPv6 []byte, numSeeders, numLeechers int) {
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

func GetStats(h Hash) (numSeeders, numLeechers int) {
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
		var seeders, leechers, swarms, seedersDeleted, leechersDeleted, swarmsDeleted int
		expiration := time.Now().Unix() - int64(duration.Seconds())
		for _, shard := range shards {
			shard.Lock()
			swarms += len(shard.swarms)
			for h, swarm := range shard.swarms {
				seeders += len(swarm.seeders)
				leechers += len(swarm.leechers)
				for peer, lastSeen := range swarm.seeders {
					if lastSeen < expiration {
						seedersDeleted++
						delete(swarm.seeders, peer)
					}
				}
				for peer, lastSeen := range swarm.leechers {
					if lastSeen < expiration {
						leechersDeleted++
						delete(swarm.leechers, peer)
					}
				}
				if len(swarm.leechers) == 0 && len(swarm.seeders) == 0 {
					swarmsDeleted++
					delete(shard.swarms, h)
				}
			}
			shard.Unlock()
		}
		log.Printf("seeders: %d (%d deleted), leechers: %d (%d deleted), swarms: %d (%d deleted)",
			seeders, seedersDeleted, leechers, leechersDeleted, swarms, swarmsDeleted)
		runtime.GC()
	}
}
