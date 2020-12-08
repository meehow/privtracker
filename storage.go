package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"net"
	"sync"
	"time"
)

type serializedPeer string
type hash [20]byte

type shard struct {
	swarms map[hash]swarm
	sync.RWMutex
}

type swarm struct {
	seeders  map[serializedPeer]int64
	leechers map[serializedPeer]int64
}

var shards = NewShards(512)
var v4InV6Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}

func shardIndex(hash [20]byte) int {
	return int(binary.BigEndian.Uint32(hash[:4])) % len(shards)
}

func NewShards(size int) []*shard {
	shards := make([]*shard, size)
	for i := 0; i < size; i++ {
		shards[i] = &shard{
			swarms: make(map[hash]swarm),
		}
	}
	return shards
}

func serialize(ip string, port uint16) serializedPeer {
	return serializedPeer(append(net.ParseIP(ip), byte(port>>8), byte(port)))
}

func PutPeer(room, infoHash, ip string, port uint16, seeding bool) {
	h := sha1.Sum([]byte(room + infoHash))
	shard := shards[shardIndex(h)]
	shard.Lock()
	if _, ok := shard.swarms[h]; !ok {
		shard.swarms[h] = swarm{
			seeders:  make(map[serializedPeer]int64),
			leechers: make(map[serializedPeer]int64),
		}
	}
	client := serialize(ip, port)
	if seeding {
		shard.swarms[h].seeders[client] = time.Now().Unix()
	} else {
		shard.swarms[h].leechers[client] = time.Now().Unix()
	}
	shard.Unlock()
}

func DeletePeer(room, infoHash, ip string, port uint16) {
	h := sha1.Sum([]byte(room + infoHash))
	shard := shards[shardIndex(h)]
	shard.Lock()
	if _, ok := shard.swarms[h]; !ok {
		return
	}
	client := serialize(ip, port)
	delete(shard.swarms[h].seeders, client)
	delete(shard.swarms[h].leechers, client)
	shard.Unlock()
}

func GraduateLeecher(room, infoHash, ip string, port uint16) {
	h := sha1.Sum([]byte(room + infoHash))
	shard := shards[shardIndex(h)]
	shard.Lock()
	if _, ok := shard.swarms[h]; !ok {
		shard.swarms[h] = swarm{
			seeders:  make(map[serializedPeer]int64),
			leechers: make(map[serializedPeer]int64),
		}
	}
	client := serialize(ip, port)
	shard.swarms[h].seeders[client] = time.Now().Unix()
	delete(shard.swarms[h].leechers, client)
	shard.Unlock()
}

func GetPeers(room, infoHash, ip string, port uint16, seeding bool, numWant uint) (peersIPv4, peersIPv6 []byte, numSeeders, numLeechers int) {
	h := sha1.Sum([]byte(room + infoHash))
	shard := shards[shardIndex(h)]
	shard.RLock()
	client := serialize(ip, port)
	// seeders don't need other seeders
	if !seeding {
		for peer := range shard.swarms[h].seeders {
			if numWant == 0 {
				break
			}
			if bytes.HasPrefix([]byte(peer), v4InV6Prefix) {
				peersIPv4 = append(peersIPv4, peer[12:]...)
			} else {
				peersIPv6 = append(peersIPv6, peer...)
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
		if bytes.HasPrefix([]byte(peer), v4InV6Prefix) {
			peersIPv4 = append(peersIPv4, peer[12:]...)
		} else {
			peersIPv6 = append(peersIPv6, peer...)
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

func Cleanup() {
	for {
		expiration := time.Now().Unix() - 600
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
		time.Sleep(time.Minute * 3)
	}
}
