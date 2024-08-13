package consulkv

import (
	"strings"

	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/records"
)

func PrepareResponseRcode(request *dns.Msg, rcode int, recursionAvailable bool) *dns.Msg {
	m := new(dns.Msg)
	m.SetRcode(request, rcode)
	m.Authoritative = true
	m.RecursionAvailable = recursionAvailable

	return m
}

func PrepareResponseReply(request *dns.Msg, recursionAvailable bool) *dns.Msg {
	m := new(dns.Msg)
	m.SetReply(request)
	m.Authoritative = true
	m.RecursionAvailable = recursionAvailable

	return m
}

func GetZoneAndRecord(zones []string, qname string) (string, string) {
	qname = strings.TrimSuffix(dns.Fqdn(qname), ".")

	for _, zone := range zones {
		if strings.HasSuffix(qname, zone) {
			record := strings.TrimSuffix(qname, zone)
			record = strings.TrimSuffix(record, ".")

			if record == "" {
				record = "@"
			}

			return zone, record
		}
	}

	return "", ""
}

func GetDefaultSOA(zoneName string) *records.SOARecord {
	return &records.SOARecord{
		MNAME:   "ns." + zoneName,
		RNAME:   "hostmaster." + zoneName,
		SERIAL:  soaSerial,
		REFRESH: 3600,
		RETRY:   600,
		EXPIRE:  86400,
		MINIMUM: 3600,
	}
}

func GetDefaultTTL(record *records.Record) int {
	if record.TTL != nil {
		return *record.TTL
	}

	return 3600 // Default TTL
}
