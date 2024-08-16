package consulkv

import (
	"encoding/json"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
)

type handler func(*api.KVPair) error

func (consul ConsulConfig) WatchConsulKey(key string, fn handler) error {
	params := map[string]interface{}{
		"type":  "key",
		"key":   consul.KVPrefix + "/" + key,
		"token": consul.Token,
	}

	watcher, err := watch.Parse(params)
	if err != nil {
		return err
	}

	watcher.Handler = func(idx uint64, raw interface{}) {
		if raw == nil {
			return
		}

		kv, ok := raw.(*api.KVPair)
		if !ok || kv == nil {
			return
		}

		fn(kv)
	}

	go func() {
		if err := watcher.Run(consul.Address); err != nil {
			logging.Log.Errorf("Error running watch plan: %v", err)
		}
	}()

	logging.Log.Infof("Started watching Consul key '%s/%s'", consul.KVPrefix, key)

	return nil
}

func (consul ConsulConfig) WatchConsulConfig(config *ConsulKVConfig) error {
	i := 0
	err := consul.WatchConsulKey("config", func(kv *api.KVPair) error {
		if i > 0 {
			if err := json.Unmarshal(kv.Value, &config); err != nil {
				logging.Log.Errorf("%s", err)

				IncrementMetricsConsulConfigUpdatedTotal("ERROR")
				return err
			}

			logging.Log.Infof("Updated Consul Config from '%s/config'", consul.KVPrefix)
			IncrementMetricsConsulConfigUpdatedTotal("NOERROR")
		}
		i++
		return nil
	})

	if err != nil {

		return err
	}

	return nil
}
