package consulkv

import (
	"encoding/json"

	"github.com/miekg/dns"
)

func (c ConsulKV) GetRecordFromConsul(key string) (*Record, error) {
	kv, _, err := c.Client.KV().Get(key, nil)
	if err != nil {
		return nil, err
	}
	if kv == nil {
		return nil, nil
	}

	var record Record
	err = json.Unmarshal(kv.Value, &record)
	if err != nil {
		log.Errorf("Error converting json: %v", kv.Value)

		return nil, err
	}

	return &record, nil
}

func (c ConsulKV) GetSOARecordFromConsul(zoneName string) (*SOARecord, error) {
	key := BuildConsulKey(c.Prefix, zoneName, "@")
	record, err := c.GetRecordFromConsul(key)
	if err != nil {
		return nil, err
	}

	if record != nil {
		for _, rec := range record.Records {
			if rec.Type == "SOA" {
				var soa SOARecord
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

func (c ConsulKV) HandleConsulError(zoneName string, r *dns.Msg, w dns.ResponseWriter, e error) (int, error) {
	m := new(dns.Msg)
	m.SetRcode(r, dns.RcodeServerFailure)
	m.Authoritative = true
	m.RecursionAvailable = true

	err := w.WriteMsg(m)
	if err != nil {
		log.Errorf("Error writing DNS error response: %v", err)

		invalidResponses.WithLabelValues(dns.Fqdn(zoneName)).Inc()

		return dns.RcodeServerFailure, err
	}

	return dns.RcodeServerFailure, e
}
