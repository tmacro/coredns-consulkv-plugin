package records

import (
	"encoding/json"
	"strings"

	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
)

func AppendPTRRecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) (bool, error) {
	var domains []string
	if err := json.Unmarshal(value, &domains); err != nil {
		return false, err
	}

	for _, domain := range domains {
		if domain != "" {
			if !IsValidDomain(domain) {
				logging.Log.Warningf("Invalid domain in PTR record: %s", domain)

			} else {
				rr := &dns.PTR{
					Hdr: dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: uint32(ttl)},
					Ptr: dns.Fqdn(domain),
				}
				msg.Answer = append(msg.Answer, rr)
			}
		}
	}

	return len(domains) > 0, nil
}

func IsValidDomain(domain string) bool {
	if !strings.Contains(domain, ".") {
		return false
	}

	for _, char := range domain {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '.') {
			return false
		}
	}

	return true
}
