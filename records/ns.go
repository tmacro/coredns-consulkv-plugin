package records

import (
	"encoding/json"

	"github.com/miekg/dns"
)

func AppendNSRecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) (bool, error) {
	var nameservers []string
	if err := json.Unmarshal(value, &nameservers); err != nil {
		return false, err
	}

	for _, ns := range nameservers {
		rr := &dns.NS{
			Hdr: dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: uint32(ttl)},
			Ns:  dns.Fqdn(ns),
		}
		msg.Answer = append(msg.Answer, rr)
	}

	return len(nameservers) > 0, nil
}
