package scw

import (
	"reflect"
	"testing"
	"time"

	"github.com/caddyserver/caddy"
)

func TestParseConfig(t *testing.T) {
	controller := caddy.NewTestController("http", `
		localhost:8080
		scw_redis_uri 127.0.0.1:6237
		scw_update_interval 15s
	`)
	actual, err := parseConfig(controller)
	if err != nil {
		t.Errorf("parseConfig return err: %v", err)
	}
	expected := Config{
		RedisURI:       "127.0.0.1.6237",
		UpdateInterval: 15 * time.Second,
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v actual %v", expected, actual)
	}
}
