package consulkv

import (
	"sync"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
	"github.com/mwantia/coredns-consulkv-plugin/types"
)

type ConsulKVPlugin struct {
	Next   plugin.Handler
	Consul *ConsulConfig
	Config *ConsulKVConfig
	cfgMu  *sync.RWMutex
}

type ConsulKVConfig struct {
	ZonePrefix  string               `json:"zone_prefix"`
	Zones       []string             `json:"zones"`
	Flattening  types.FlatteningType `json:"flattening,omitempty"`
	NoCache     bool                 `json:"no_cache,omitempty"`
	ConsulCache *ConsulKVCache       `json:"consul_cache,omitempty"`
}

type ConsulKVCache struct {
	UseCache   *bool `json:"use_cache,omitempty"`
	MaxAge     *int  `json:"max_age"`
	Consistent *bool `json:"consistent"`
	AllowStale *bool `json:"allowstale"`
}

func CreatePlugin(c *caddy.Controller) (*ConsulKVPlugin, error) {
	plug := &ConsulKVPlugin{
		cfgMu: new(sync.RWMutex),
	}

	consul, err := CreateConsulConfig(c)
	if err != nil {
		return nil, err
	}

	config, err := consul.GetConfigFromConsul()
	if err != nil {
		return nil, err
	}

	plug.Consul = consul
	plug.Config = config

	return plug, nil
}
