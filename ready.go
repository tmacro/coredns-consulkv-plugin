package consulkv

import (
	"time"

	"github.com/hashicorp/consul/api"
)

func (config ConsulKVPlugin) Ready() bool {
	if config.Consul == nil {
		return false
	}

	_, _, err := config.Consul.Client.Health().Service("consul", "", false, &api.QueryOptions{
		AllowStale:        true,
		UseCache:          true,
		MaxAge:            1 * time.Second,
		RequireConsistent: false,
	})

	return err == nil
}
