package records

import (
	"strings"

	"github.com/miekg/dns"
)

type SOARecord struct {
	MNAME   string `json:"mname"`
	RNAME   string `json:"rname"`
	SERIAL  uint32 `json:"serial"`
	REFRESH uint32 `json:"refresh"`
	RETRY   uint32 `json:"retry"`
	EXPIRE  uint32 `json:"expire"`
	MINIMUM uint32 `json:"minimum"`
}

func AppendSOAToAuthority(msg *dns.Msg, qname string, soa *SOARecord) {
	rr := &dns.SOA{
		Hdr:     dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: soa.MINIMUM},
		Ns:      dns.Fqdn(soa.MNAME),
		Mbox:    dns.Fqdn(strings.Replace(soa.RNAME, "@", ".", 1)),
		Serial:  soa.SERIAL,
		Refresh: soa.REFRESH,
		Retry:   soa.RETRY,
		Expire:  soa.EXPIRE,
		Minttl:  soa.MINIMUM,
	}
	msg.Ns = append(msg.Ns, rr)
}

func AppendSOARecord(msg *dns.Msg, qname string, soa *SOARecord) bool {
	rr := &dns.SOA{
		Hdr:     dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: soa.MINIMUM},
		Ns:      dns.Fqdn(soa.MNAME),
		Mbox:    dns.Fqdn(soa.RNAME),
		Serial:  soa.SERIAL,
		Refresh: soa.REFRESH,
		Retry:   soa.RETRY,
		Expire:  soa.EXPIRE,
		Minttl:  soa.MINIMUM,
	}
	msg.Answer = append(msg.Answer, rr)

	return true
}
