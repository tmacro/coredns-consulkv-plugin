package records

import (
	"encoding/json"
	"net"

	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
)

func AppendAAAARecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) bool {
	var ips []string
	if err := json.Unmarshal(value, &ips); err != nil {
		logging.Log.Errorf("Error parsing JSON for AAAA record: %v", err)
		return false
	}

	for _, ip := range ips {
		rr := &dns.AAAA{
			Hdr:  dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: uint32(ttl)},
			AAAA: net.ParseIP(ip),
		}
		msg.Answer = append(msg.Answer, rr)
	}

	return len(ips) > 0
}
