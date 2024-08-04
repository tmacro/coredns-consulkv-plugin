package consulkv

import (
	"time"

	"github.com/hashicorp/consul/api"
)

func (c ConsulKV) Ready() bool {
	if c.Client == nil {
		return false
	}

	_, _, err := c.Client.Health().Service("consul", "", false, &api.QueryOptions{
		AllowStale:        true,
		UseCache:          true,
		MaxAge:            1 * time.Second,
		RequireConsistent: false,
	})

	return err == nil
}
