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

func (c ConsulKV) GetZoneAndRecordName(qname string) (string, string) {
	qname = strings.TrimSuffix(dns.Fqdn(qname), ".")

	for _, zone := range c.Zones {
		if strings.HasSuffix(qname, zone) {
			recordName := strings.TrimSuffix(qname, zone)
			recordName = strings.TrimSuffix(recordName, ".")

			if recordName == "" {
				recordName = "@"
			}

			return zone, recordName
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
