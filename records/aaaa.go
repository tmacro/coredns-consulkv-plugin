package records

import (
	"encoding/json"
	"net"

	"github.com/miekg/dns"
)

func AppendAAAARecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) (bool, error) {
	var ips []string
	if err := json.Unmarshal(value, &ips); err != nil {
		return false, err
	}

	for _, ip := range ips {
		rr := &dns.AAAA{
			Hdr:  dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: uint32(ttl)},
			AAAA: net.ParseIP(ip),
		}
		msg.Answer = append(msg.Answer, rr)
	}

	return len(ips) > 0, nil
}
