package consulkv

import (
	"strings"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
	"github.com/hashicorp/consul/api"
	"github.com/miekg/dns"
)

type FlatteningType string

const (
	Flattening_None  FlatteningType = "none"
	Flattening_Local FlatteningType = "local"
	Flattening_Full  FlatteningType = "full"
)

type ConsulKV struct {
	Next       plugin.Handler
	Client     *api.Client
	Prefix     string
	Address    string
	Token      string
	Zones      []string
	NoCache    bool
	Flattening FlatteningType
}

func CreateConfig() *ConsulKV {
	conf := &ConsulKV{
		Prefix:     "dns",
		Address:    "http://127.0.0.1:8500",
		Zones:      []string{},
		Flattening: Flattening_Local,
		NoCache:    false,
	}

	return conf
}

func (conf ConsulKV) GetZoneAndRecord(qname string) (string, string) {
	qname = strings.TrimSuffix(dns.Fqdn(qname), ".")

	for _, zone := range conf.Zones {
		if strings.HasSuffix(qname, zone) {
			record := strings.TrimSuffix(qname, zone)
			record = strings.TrimSuffix(record, ".")

			if record == "" {
				record = "@"
			}

			return zone, record
		}
	}

	return "", ""
}

func LoadCorefile(conf *ConsulKV, c *caddy.Controller) error {
	for c.Next() {
		if c.NextBlock() {
			for {
				switch c.Val() {
				case "prefix":
					if !c.NextArg() {
						return c.ArgErr()
					}
					conf.Prefix = c.Val()

				case "address":
					if !c.NextArg() {
						return c.ArgErr()
					}
					conf.Address = c.Val()

				case "token":
					if !c.NextArg() {
						return c.ArgErr()
					}
					conf.Token = c.Val()

				case "zones":
					args := c.RemainingArgs()
					if len(args) == 0 {
						return c.ArgErr()
					}
					conf.Zones = append(conf.Zones, args...)

				case "flattening":
					if !c.NextArg() {
						return c.ArgErr()
					}

					flatteningtype := c.Val()
					switch FlatteningType(flatteningtype) {
					case Flattening_None, Flattening_Local, Flattening_Full:
						conf.Flattening = FlatteningType(flatteningtype)

					default:
						return c.Errf("invalid flattening mode: %s", flatteningtype)
					}

				case "no_cache":
					conf.NoCache = true

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

	return nil
}
