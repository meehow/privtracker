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

type Hash [sha1.Size]byte // we use sha1 and we are not affraid of hash collisions
type Peer [18]byte        // 16 bytes for IP and 2 bytes for port number

var v4InV6Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}
var shards [256]*shard

type shard struct {
	swarms map[Hash]*Swarm
	sync.RWMutex
}

type Swarm struct {
	seeders  map[Peer]int64
	leechers map[Peer]int64
}

func init() {
	for i := range shards {
		shards[i] = &shard{
			swarms: make(map[Hash]*Swarm),
		}
	}
	go Cleanup(time.Minute * 16) // needs to be bigget than biggest interval in announce.go
}

func NewSwarm() *Swarm {
	return &Swarm{
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

func shardIndex(hash Hash) uint8 {
	// return int(binary.BigEndian.Uint16(hash[:2])) % len(shards)
	return hash[0] // works only with 256 shards
}

func PutPeer(h Hash, peer Peer, seeding bool) {
	shard := shards[shardIndex(h)]
	shard.Lock()
	defer shard.Unlock()
	if _, ok := shard.swarms[h]; !ok {
		shard.swarms[h] = NewSwarm()
	}
	if seeding {
		shard.swarms[h].seeders[peer] = time.Now().Unix()
	} else {
		shard.swarms[h].leechers[peer] = time.Now().Unix()
	}
}

func DeletePeer(h Hash, peer Peer) {
	shard := shards[shardIndex(h)]
	shard.Lock()
	defer shard.Unlock()
	if _, ok := shard.swarms[h]; !ok {
		return
	}
	delete(shard.swarms[h].seeders, peer)
	delete(shard.swarms[h].leechers, peer)
}

func GraduateLeecher(h Hash, peer Peer) {
	shard := shards[shardIndex(h)]
	shard.Lock()
	defer shard.Unlock()
	if _, ok := shard.swarms[h]; !ok {
		shard.swarms[h] = NewSwarm()
	}
	shard.swarms[h].seeders[peer] = time.Now().Unix()
	delete(shard.swarms[h].leechers, peer)
}

func GetPeers(h Hash, client Peer, seeding bool, numWant int) (peersIPv4, peersIPv6 []byte, numSeeders, numLeechers int) {
	shard := shards[shardIndex(h)]
	shard.RLock()
	defer shard.RUnlock()
	if _, ok := shard.swarms[h]; !ok {
		return
	}

	numSeeders = len(shard.swarms[h].seeders)
	numLeechers = len(shard.swarms[h].leechers)

	// seeders don't need other seeders
	if !seeding {
		for peer := range shard.swarms[h].seeders {
			if numWant == 0 {
				break
			}
			if peer == client {
				continue
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
		if numWant == 0 {
			break
		}
		if peer == client {
			continue
		}
		if bytes.HasPrefix(peer[:], v4InV6Prefix) {
			peersIPv4 = append(peersIPv4, peer[len(v4InV6Prefix):]...)
		} else {
			peersIPv6 = append(peersIPv6, peer[:]...)
		}
		numWant--
	}
	return
}

func GetStats(h Hash) (numSeeders, numLeechers int) {
	shard := shards[shardIndex(h)]
	shard.RLock()
	defer shard.RUnlock()
	if _, ok := shard.swarms[h]; !ok {
		return
	}
	numSeeders = len(shard.swarms[h].seeders)
	numLeechers = len(shard.swarms[h].leechers)
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
