package consulkv

import (
	"strings"

	"github.com/miekg/dns"
)

func (c ConsulKV) GetZoneAndRecordName(qname string) (string, string) {
	// Remove the trailing dot if present
	qname = strings.TrimSuffix(dns.Fqdn(qname), ".")
	// Find the matching zone
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

func GetRecordTTL(record *Record) int {
	if record.TTL != nil {
		return *record.TTL
	}

	return 3600 // Default TTL
}
