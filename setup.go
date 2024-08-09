package consulkv

import (
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
)

var soaSerial = uint32(time.Now().Unix())

func init() {
	plugin.Register("consulkv", setup)
}

func setup(c *caddy.Controller) error {
	c.OnStartup(func() error {
		prometheus.MustRegister(successfulQueries)
		prometheus.MustRegister(failedQueries)
		prometheus.MustRegister(consulErrors)
		prometheus.MustRegister(invalidResponses)

		return nil
	})

	conf := CreateConfig()
	LoadCorefile(conf, c)

	err := conf.CreateConsulClient()
	if err != nil {
		return plugin.Error("consulkv", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		conf.Next = next

		return conf
	})

	return nil
}
