package consulkv

import (
	"encoding/json"
	"os"
	"time"

	"github.com/coredns/caddy"
	"github.com/hashicorp/consul/api"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
	"github.com/mwantia/coredns-consulkv-plugin/records"
)

type ConsulConfig struct {
	Client       *api.Client
	KVPrefix     string
	Address      string
	Token        string
	DisableWatch bool
}

func GetConsulEnvConfig() ConsulConfig {
	return ConsulConfig{
		Address:  os.Getenv("CONSUL_HTTP_ADDR"),
		Token:    os.Getenv("CONSUL_HTTP_TOKEN"),
		KVPrefix: os.Getenv("CONSUL_KV_PREFIX"),
	}
}

func CreateConsulConfig(c *caddy.Controller) (*ConsulConfig, error) {
	consul := &ConsulConfig{
		KVPrefix: "dns",
		Address:  "http://127.0.0.1:8500",
		Token:    "",
	}

	err := LoadConsulConfig(c, consul)
	if err != nil {
		return nil, err
	}

	env := GetConsulEnvConfig()

	if env.Address != "" {
		consul.Address = env.Address
	}
	if env.Token != "" {
		consul.Token = env.Token
	}
	if env.KVPrefix != "" {
		consul.KVPrefix = env.KVPrefix
	}

	err = CreateConsulClient(consul)
	if err != nil {
		return nil, err
	}

	return consul, nil
}

func LoadConsulConfig(c *caddy.Controller, consul *ConsulConfig) error {
	n := 0
	for c.Next() {
		if n > 0 {
			return c.Err("Unable to load config")
		}
		n++

		args := c.RemainingArgs()
		if len(args) >= 1 {
			for i := range args {
				LoadEnvFile(args[i])
			}
		}

		for c.NextBlock() {
			val := c.Val()
			args = c.RemainingArgs()

			switch val {

			case "address":
				if len(args) < 1 {
					return c.Errf("config 'address' can't be empty")
				}
				consul.Address = args[0]

			case "token":
				if len(args) < 1 {
					return c.Errf("config 'token' can't be empty")
				}
				consul.Token = args[0]

			case "kv_prefix":
				if len(args) < 1 {
					return c.Errf("config 'kv_prefix' can't be empty")
				}
				consul.KVPrefix = args[0]

			case "disable_watch":
				consul.DisableWatch = true
			}
		}
	}

	return nil
}

func CreateConsulClient(consul *ConsulConfig) error {
	def := api.DefaultConfig()
	def.Address = consul.Address
	def.Token = consul.Token

	client, err := api.NewClient(def)
	if err != nil {
		return err
	}

	consul.Client = client
	return nil
}

func (consul *ConsulConfig) GetConsulKeyValue(key string, cache *ConsulKVCache) (*api.KVPair, float64, error) {
	logging.Log.Debugf("Constructed key: '%s'", consul.KVPrefix+"/"+key)

	start := time.Now()
	options := CreateQueryOptions(cache)
	kv, _, err := consul.Client.KV().Get(consul.KVPrefix+"/"+key, options)
	duration := time.Since(start).Seconds()

	return kv, duration, err
}

func (consul *ConsulConfig) GetConfigFromConsul() (*ConsulKVConfig, error) {
	kv, duration, err := consul.GetConsulKeyValue("config", nil)

	if err != nil {
		IncrementMetricsConsulRequestDurationSeconds("ERROR", duration)
		return nil, err
	}

	if kv == nil {
		IncrementMetricsConsulRequestDurationSeconds("NODATA", duration)
		return nil, err
	}

	var config ConsulKVConfig
	err = json.Unmarshal(kv.Value, &config)

	if err != nil {
		logging.Log.Errorf("Error converting json: %v", kv.Key)
		IncrementMetricsConsulRequestDurationSeconds("ERROR", duration)
		return nil, err
	}

	IncrementMetricsConsulRequestDurationSeconds("NOERROR", duration)
	return &config, nil
}

func (consul ConsulConfig) GetZoneRecordFromConsul(zone, name string, cache *ConsulKVCache) (*records.Record, error) {
	kv, duration, err := consul.GetConsulKeyValue("zones/"+zone+"/"+name, cache)

	if err != nil {
		IncrementMetricsConsulRequestDurationSeconds("ERROR", duration)
		return nil, err
	}

	if kv == nil {
		IncrementMetricsConsulRequestDurationSeconds("NODATA", duration)
		return nil, err
	}

	var record records.Record
	err = json.Unmarshal(kv.Value, &record)
	if err != nil {
		logging.Log.Errorf("Error converting json: %v", kv.Value)
		IncrementMetricsConsulRequestDurationSeconds("ERROR", duration)
		return nil, err
	}

	IncrementMetricsConsulRequestDurationSeconds("NOERROR", duration)
	return &record, nil
}

func (consul ConsulConfig) GetSOARecordFromConsul(zone string, cache *ConsulKVCache) (*records.SOARecord, error) {
	record, err := consul.GetZoneRecordFromConsul(zone, "@", cache)

	if record != nil {
		for _, rec := range record.Records {
			if rec.Type == "SOA" {
				var soa records.SOARecord
				err = json.Unmarshal(rec.Value, &soa)
				if err != nil {
					return nil, err
				}

				return &soa, nil
			}
		}
	}

	return GetDefaultSOA(zone), err
}

func CreateQueryOptions(cache *ConsulKVCache) *api.QueryOptions {
	options := &api.QueryOptions{
		UseCache:          true,
		MaxAge:            time.Minute,
		StaleIfError:      10 * time.Second,
		RequireConsistent: false,
		AllowStale:        true,
	}

	if cache != nil {
		if cache.UseCache != nil {
			options.UseCache = *cache.UseCache
		}
		if cache.MaxAge != nil {
			options.MaxAge = time.Duration(*cache.MaxAge)
		}
		if cache.Consistent != nil {
			options.RequireConsistent = *cache.Consistent
		}
		if cache.AllowStale != nil {
			options.AllowStale = *cache.AllowStale
		}
	}

	return options
}
