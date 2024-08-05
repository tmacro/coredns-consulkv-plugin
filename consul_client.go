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
		return nil, err
	}

	return &record, nil
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
