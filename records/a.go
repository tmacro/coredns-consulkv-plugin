package records

import (
	"encoding/json"
	"net"

	"github.com/miekg/dns"
)

func AppendARecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) (bool, error) {
	var ips []string
	if err := json.Unmarshal(value, &ips); err != nil {
		return false, err
	}

	for _, ip := range ips {
		rr := &dns.A{
			Hdr: dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: uint32(ttl)},
			A:   net.ParseIP(ip),
		}
		msg.Answer = append(msg.Answer, rr)
	}

	return len(ips) > 0, nil
}
