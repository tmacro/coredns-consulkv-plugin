package records

import (
	"encoding/json"

	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
)

type SRVRecord struct {
	Target   string `json:"target"`
	Port     uint16 `json:"port"`
	Priority uint16 `json:"priority"`
	Weight   uint16 `json:"weight"`
}

func AppendSRVRecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) bool {
	var records []SRVRecord
	if err := json.Unmarshal(value, &records); err != nil {
		logging.Log.Errorf("Error parsing JSON for SRV record: %v", err)
		return false
	}

	for _, record := range records {
		rr := &dns.SRV{
			Hdr:      dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: uint32(ttl)},
			Priority: record.Priority,
			Weight:   record.Weight,
			Port:     record.Port,
			Target:   dns.Fqdn(record.Target),
		}
		msg.Answer = append(msg.Answer, rr)
	}

	return len(records) > 0
}
