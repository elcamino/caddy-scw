package scw

import (
	"time"

	"github.com/caddyserver/caddy"
)

// Config specifies configuration parsed for Caddyfile
type Config struct {
	RedisURI       string
	UpdateInterval time.Duration
}

func parseConfig(c *caddy.Controller) (Config, error) {
	var config = Config{}
	for c.Next() {
		value := c.Val()
		switch value {
		case "scw_redis_uri":
			if !c.NextArg() {
				continue
			}
			config.RedisURI = c.Val()
		case "scw_update_interval":
			if !c.NextArg() {
				continue
			}

			config.UpdateInterval, _ = time.ParseDuration(c.Val())
		}
	}
	return config, nil
}
