package records

import (
	"encoding/json"
	"strings"

	"github.com/miekg/dns"
)

func AppendDnsSdPTRRecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) (bool, error) {
	var services []string
	if err := json.Unmarshal(value, &services); err != nil {
		return false, err
	}

	for _, service := range services {
		rr := &dns.PTR{
			Hdr: dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: uint32(ttl)},
			Ptr: dns.Fqdn(service),
		}
		msg.Answer = append(msg.Answer, rr)
	}

	return len(services) > 0, nil
}

func IsDnsSdQuery(qname string) bool {
	return strings.HasSuffix(qname, "._dns-sd._udp.") || strings.HasSuffix(qname, "._tcp.") || strings.HasSuffix(qname, "._udp.")
}
