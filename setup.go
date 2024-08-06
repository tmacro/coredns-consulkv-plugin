package consulkv

import (
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/hashicorp/consul/api"
	"github.com/prometheus/client_golang/prometheus"
)

var pluginname = "consulkv"
var soaSerial = uint32(time.Now().Unix())
var log = clog.NewWithPlugin(pluginname)

func init() {
	plugin.Register(pluginname, setup)
}

func setup(c *caddy.Controller) error {
	p := &ConsulKV{
		Prefix:  "dns",
		Address: "http://127.0.0.1:8500",
		Zones:   []string{},
	}

	for c.Next() {
		if c.NextBlock() {
			for {
				switch c.Val() {
				case "prefix":
					if !c.NextArg() {
						return c.ArgErr()
					}
					p.Prefix = c.Val()
				case "address":
					if !c.NextArg() {
						return c.ArgErr()
					}
					p.Address = c.Val()
				case "token":
					if !c.NextArg() {
						return c.ArgErr()
					}
					p.Token = c.Val()
				case "zones":
					args := c.RemainingArgs()
					if len(args) == 0 {
						return c.ArgErr()
					}
					p.Zones = append(p.Zones, args...)
				default:
					if c.Val() != "}" {
						return c.Errf("unknown property '%s'", c.Val())
					}
				}
				if !c.Next() {
					break
				}
			}
		}
	}

	c.OnStartup(func() error {
		prometheus.MustRegister(successfulQueries)
		prometheus.MustRegister(failedQueries)
		prometheus.MustRegister(consulErrors)
		prometheus.MustRegister(invalidResponses)

		return nil
	})

	config := api.DefaultConfig()
	config.Address = p.Address
	config.Token = p.Token
	client, err := api.NewClient(config)

	if err != nil {
		return plugin.Error(pluginname, err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		p.Next = next
		p.Client = client

		return p
	})

	return nil
}
