package consulkv

import (
	"strings"

	"github.com/miekg/dns"
)

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

func BuildConsulKey(prefix, zone, record string) string {
	return prefix + "/" + zone + "/" + record
}

func GetDefaultSOA(zoneName string) *SOARecord {
	return &SOARecord{
		MNAME:   "ns." + zoneName,
		RNAME:   "hostmaster." + zoneName,
		SERIAL:  soaSerial,
		REFRESH: 3600,
		RETRY:   600,
		EXPIRE:  86400,
		MINIMUM: 3600,
	}
}

func GetDefaultTTL(record *Record) int {
	if record.TTL != nil {
		return *record.TTL
	}

	return 3600 // Default TTL
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
