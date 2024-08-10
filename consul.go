package consulkv

import (
	"encoding/json"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
	"github.com/mwantia/coredns-consulkv-plugin/records"
)

func (conf *ConsulKV) CreateConsulClient() error {
	def := api.DefaultConfig()
	def.Address = conf.Address
	def.Token = conf.Token

	client, err := api.NewClient(def)
	if err != nil {
		return nil
	}

	conf.Client = client
	return nil
}

func (conf ConsulKV) BuildConsulKey(zone, record string) string {
	return conf.Prefix + "/" + zone + "/" + record
}

func (conf ConsulKV) GetRecordFromConsul(key string) (*records.Record, error) {
	start := time.Now()
	kv, _, err := conf.Client.KV().Get(key, nil)
	duration := time.Since(start).Seconds()

	if err != nil {
		IncrementMetricsConsulRequestDurationSeconds("ERROR", duration)

		return nil, err
	}
	if kv == nil {
		IncrementMetricsConsulRequestDurationSeconds("NODATA", duration)

		return nil, nil
	}

	var record records.Record
	err = json.Unmarshal(kv.Value, &record)
	if err != nil {
		logging.Log.Errorf("Error converting json: %v", kv.Value)
		IncrementMetricsConsulRequestDurationSeconds("", duration)

		return nil, err
	}

	IncrementMetricsConsulRequestDurationSeconds("NOERROR", duration)

	return &record, nil
}

func (conf ConsulKV) GetSOARecordFromConsul(zoneName string) (*records.SOARecord, error) {
	key := conf.BuildConsulKey(zoneName, "@")
	record, err := conf.GetRecordFromConsul(key)
	if err != nil {
		return nil, err
	}

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

	return GetDefaultSOA(zoneName), nil
}

func HandleConsulError(zoneName string, r *dns.Msg, w dns.ResponseWriter, e error) (int, error) {
	m := new(dns.Msg)
	m.SetRcode(r, dns.RcodeServerFailure)
	m.Authoritative = true
	m.RecursionAvailable = true

	err := w.WriteMsg(m)
	if err != nil {
		logging.Log.Errorf("Error writing DNS error response: %v", err)
		return dns.RcodeServerFailure, err
	}

	return dns.RcodeServerFailure, e
}
