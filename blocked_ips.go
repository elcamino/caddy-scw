package scw

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v7"
)

// BlockedIPs contains all currently blocked IP addresses
type BlockedIPs struct {
	r *redis.Client

	redisAddress   string
	updateInterval time.Duration
	ips            map[string]bool
	ipsMutex       sync.RWMutex
}

// NewBlockedIPs creates a new BlockedIPs item
func NewBlockedIPs(redisAddr string, updateInterval time.Duration) (*BlockedIPs, error) {
	blip := BlockedIPs{
		r: redis.NewClient(&redis.Options{
			Addr: redisAddr,
			// Password: "", // no password set
			DB: 0, // use default DB
		}),
		redisAddress:   redisAddr,
		ips:            make(map[string]bool),
		ipsMutex:       sync.RWMutex{},
		updateInterval: updateInterval,
	}

	pong := blip.r.Ping()
	if err := pong.Err(); err != nil {
		return nil, err
	}

	go blip.loadBlacklistedIPs()

	return &blip, nil
}

func (b *BlockedIPs) loadBlacklistedIPs() {
	for {
		// startTs := time.Now()
		sleep := time.After(b.updateInterval)

		ips := make(map[string]bool)

		client := redis.NewClient(&redis.Options{
			Addr: b.redisAddress,
			// Password: "", // no password set
			DB: 0, // use default DB
		})

		_, err := client.Ping().Result()
		if err != nil {
			log.Printf("Failed to connect to Redis at %s: %s", b.redisAddress, err)
		} else {

			blocked, err := client.Keys("bl:*").Result()
			if err == nil {
				for _, ip := range blocked {
					ips[ip[3:]] = true
				}
				// traceLog("pulled blacklist fron redis in %v\n", time.Now().Sub(startTs))
			} else {
				log.Printf("failed to pull blacklist from redis: %s\n", err)
			}

		}

		b.ipsMutex.Lock()
		b.ips = ips
		b.ipsMutex.Unlock()
		client.Close()

		<-sleep
	}
}

// IsBlocked checks whether an IP is blocked
func (b *BlockedIPs) IsBlocked(ip string, cached bool) bool {
	if cached {
		b.ipsMutex.RLock()
		defer b.ipsMutex.RUnlock()

		_, ok := b.ips[ip]
		return ok
	}

	val, err := b.r.Get(fmt.Sprintf("bl:%s", ip)).Result()
	if err != nil {
		return false
	}

	if val == "1" {
		return true
	}

	return false
}
