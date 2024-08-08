package consulkv

import (
	"encoding/json"

	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
	"github.com/mwantia/coredns-consulkv-plugin/records"
)

func (c ConsulKV) HandleRecord(msg *dns.Msg, qname string, qtype uint16, record *Record) bool {
	ttl := GetDefaultTTL(record)
	foundRequestedType := false

	logging.Log.Debugf("Amount of available records: %v", len(record.Records))

	zoneName, _ := c.GetZoneAndRecordName(qname)
	soa, err := c.GetSOARecordFromConsul(zoneName)

	if err != nil {
		logging.Log.Errorf("Error loading SOA record: %v", err)
		invalidResponses.WithLabelValues(zoneName).Inc()
	}

	for _, rec := range record.Records {
		logging.Log.Debugf("Searching record for type %s", rec.Type)

		switch rec.Type {
		case "NS":
			if qtype == dns.TypeNS {
				foundRequestedType = records.AppendNSRecords(msg, qname, ttl, rec.Value)
			}
		case "SVCB":
			if qtype == dns.TypeSVCB {
				foundRequestedType = records.AppendSVCBRecords(msg, qname, ttl, rec.Value, dns.TypeSVCB)
			}
		case "HTTPS":
			if qtype == dns.TypeHTTPS {
				foundRequestedType = records.AppendSVCBRecords(msg, qname, ttl, rec.Value, dns.TypeHTTPS)
			}
		case "SOA":
			if qtype == dns.TypeSOA || qtype == dns.TypeANY {
				foundRequestedType = records.AppendSOARecord(msg, qname, soa)
			}
		case "A":
			if qtype == dns.TypeA || (qtype == dns.TypeHTTPS && !foundRequestedType) {
				foundRequestedType = records.AppendARecords(msg, qname, ttl, rec.Value)
			}
		case "AAAA":
			if qtype == dns.TypeAAAA || (qtype == dns.TypeHTTPS && !foundRequestedType) {
				foundRequestedType = records.AppendAAAARecords(msg, qname, ttl, rec.Value)
			}
		case "CNAME":
			if qtype == dns.TypeCNAME || qtype == dns.TypeA || qtype == dns.TypeAAAA || (qtype == dns.TypeHTTPS && !foundRequestedType) {
				foundRequestedType = c.AppendCNAMERecords(msg, qname, qtype, ttl, rec.Value)
			}
		case "PTR":
			if qtype == dns.TypePTR {
				foundRequestedType = records.AppendPTRRecords(msg, qname, ttl, rec.Value)
			}

		case "SRV":
			if qtype == dns.TypeSRV {
				foundRequestedType = records.AppendSRVRecords(msg, qname, ttl, rec.Value)
			}

		case "TXT":
			txtAnswered := records.AppendTXTRecords(msg, qtype, qname, ttl, rec.Value)
			if txtAnswered {
				foundRequestedType = txtAnswered
			}
		}
	}

	if (qtype == dns.TypeSVCB || qtype == dns.TypeHTTPS) && !foundRequestedType && len(msg.Answer) > 0 {
		foundRequestedType = true
	}

	if !foundRequestedType && soa != nil && qtype != dns.TypeSOA && qtype != dns.TypeANY {
		records.AppendSOAToAuthority(msg, qname, soa)
	}

	return foundRequestedType
}

func (c *ConsulKV) AppendCNAMERecords(msg *dns.Msg, qname string, qtype uint16, ttl int, value json.RawMessage) bool {
	var alias string
	if err := json.Unmarshal(value, &alias); err != nil {
		logging.Log.Errorf("Error parsing JSON for CNAME record: %v", err)
		return false
	}

	rr := &dns.CNAME{
		Hdr:    dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: uint32(ttl)},
		Target: dns.Fqdn(alias),
	}
	msg.Answer = append(msg.Answer, rr)

	zoneName, recordName := c.GetZoneAndRecordName(alias)
	if zoneName == "" {
		logging.Log.Debugf("Zone %s not in configured zones %s, passing to next plugin", zoneName, c.Zones)
	}

	logging.Log.Debugf("Record: %s, Zone: %s", recordName, zoneName)

	key := BuildConsulKey(c.Prefix, zoneName, recordName)

	logging.Log.Debugf("Constructed key: %s", key)

	record, err := c.GetRecordFromConsul(key)
	if err != nil {
		logging.Log.Errorf("Error resolving CNAME alias %s: %v", recordName, err)
		return true
	}

	if record != nil {
		return c.HandleRecord(msg, recordName, qtype, record)
	}

	logging.Log.Debugf("No record found for alias %s and type %s", recordName, dns.TypeToString[qtype])

	return true
}
