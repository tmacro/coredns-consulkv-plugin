package consulkv

import (
	"encoding/json"

	"github.com/coredns/coredns/plugin"
	"github.com/hashicorp/consul/api"
)

type ConsulKV struct {
	Next    plugin.Handler
	Client  *api.Client
	Prefix  string
	Address string
	Token   string
	Zones   []string
}

type Record struct {
	TTL     *int `json:"ttl"`
	Records []struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	} `json:"records"`
}
