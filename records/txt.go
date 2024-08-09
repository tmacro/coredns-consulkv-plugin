package records

import (
	"encoding/json"

	"github.com/miekg/dns"
)

func AppendTXTRecords(msg *dns.Msg, qtype uint16, qname string, ttl int, value json.RawMessage) (bool, error) {
	var values []string
	if err := json.Unmarshal(value, &values); err != nil {
		return false, err
	}

	txtAnswered := false

	rr := &dns.TXT{
		Hdr: dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: uint32(ttl)},
		Txt: values,
	}

	if qtype == dns.TypeTXT {
		msg.Answer = append(msg.Answer, rr)
		txtAnswered = true
	} else {
		msg.Extra = append(msg.Extra, rr)
	}

	return txtAnswered, nil
}
