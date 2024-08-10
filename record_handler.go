package consulkv

import (
	"encoding/json"

	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
	"github.com/mwantia/coredns-consulkv-plugin/records"
)

func (c ConsulKV) HandleRecord(msg *dns.Msg, qname string, qtype uint16, record *records.Record) bool {
	ttl := GetDefaultTTL(record)
	foundRequestedType := false

	logging.Log.Debugf("Amount of available records: %v", len(record.Records))

	zname, _ := c.GetZoneAndRecord(qname)
	soa, err := c.GetSOARecordFromConsul(zname)

	if err != nil {
		logging.Log.Errorf("Error loading SOA record: %v", err)
		IncrementMetricsPluginErrorsTotal("SOA_GET")
	}

	for _, rec := range record.Records {
		logging.Log.Debugf("Searching record for type %s", rec.Type)

		switch rec.Type {
		case "NS":
			if qtype == dns.TypeNS {
				found, err := records.AppendNSRecords(msg, qname, ttl, rec.Value)
				if err != nil {
					logging.Log.Errorf("Error parsing JSON for NS record: %v", err)
					IncrementMetricsPluginErrorsTotal("JSON_UNMARSHAL")
				}

				foundRequestedType = found
			}

		case "SVCB":
			if qtype == dns.TypeSVCB {
				found, err := records.AppendSVCBRecords(msg, qname, ttl, rec.Value, dns.TypeSVCB)
				if err != nil {
					logging.Log.Errorf("Error parsing JSON for SVCB record: %v", err)
					IncrementMetricsPluginErrorsTotal("JSON_UNMARSHAL")
				}

				foundRequestedType = found
			}

		case "HTTPS":
			if qtype == dns.TypeHTTPS {
				found, err := records.AppendSVCBRecords(msg, qname, ttl, rec.Value, dns.TypeHTTPS)
				if err != nil {
					logging.Log.Errorf("Error parsing JSON for HTTPS record: %v", err)
					IncrementMetricsPluginErrorsTotal("JSON_UNMARSHAL")
				}

				foundRequestedType = found
			}

		case "SOA":
			if qtype == dns.TypeSOA || qtype == dns.TypeANY {
				foundRequestedType = records.AppendSOARecord(msg, qname, soa)
			}

		case "A":
			if qtype == dns.TypeA || (qtype == dns.TypeHTTPS && !foundRequestedType) {
				found, err := records.AppendARecords(msg, qname, ttl, rec.Value)
				if err != nil {
					logging.Log.Errorf("Error parsing JSON for A record: %v", err)
					IncrementMetricsPluginErrorsTotal("JSON_UNMARSHAL")
				}

				foundRequestedType = found
			}

		case "AAAA":
			if qtype == dns.TypeAAAA || (qtype == dns.TypeHTTPS && !foundRequestedType) {
				found, err := records.AppendAAAARecords(msg, qname, ttl, rec.Value)
				if err != nil {
					logging.Log.Errorf("Error parsing JSON for AAAA record: %v", err)
					IncrementMetricsPluginErrorsTotal("JSON_UNMARSHAL")
				}

				foundRequestedType = found
			}

		case "CNAME":
			if qtype == dns.TypeCNAME || qtype == dns.TypeA || qtype == dns.TypeAAAA || (qtype == dns.TypeHTTPS && !foundRequestedType) {
				foundRequestedType = c.AppendCNAMERecords(msg, qname, qtype, ttl, rec.Value)
			}

		case "PTR":
			if qtype == dns.TypePTR {
				found, err := records.AppendPTRRecords(msg, qname, ttl, rec.Value)
				if err != nil {
					logging.Log.Errorf("Error parsing JSON for PTR record: %v", err)
					IncrementMetricsPluginErrorsTotal("JSON_UNMARSHAL")
				}

				foundRequestedType = found
			}

		case "SRV":
			if qtype == dns.TypeSRV {
				found, err := records.AppendSRVRecords(msg, qname, ttl, rec.Value)
				if err != nil {
					logging.Log.Errorf("Error parsing JSON for SRV record: %v", err)
					IncrementMetricsPluginErrorsTotal("JSON_UNMARSHAL")
				}

				foundRequestedType = found
			}

		case "TXT":
			txtAnswered, err := records.AppendTXTRecords(msg, qtype, qname, ttl, rec.Value)
			if err != nil {
				logging.Log.Errorf("Error parsing JSON for TXT record: %v", err)
				IncrementMetricsPluginErrorsTotal("JSON_UNMARSHAL")
			}

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
		IncrementMetricsPluginErrorsTotal("JSON_UNMARSHAL")

		return false
	}

	rr := &dns.CNAME{
		Hdr:    dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: uint32(ttl)},
		Target: dns.Fqdn(alias),
	}
	msg.Answer = append(msg.Answer, rr)

	zname, rname := c.GetZoneAndRecord(alias)
	if zname == "" {
		logging.Log.Debugf("Zone %s not in configured zones %s, skipping alias lookup", zname, c.Zones)
	}

	logging.Log.Debugf("Received new request for zone '%s' and record '%s' with code '%s", zname, rname, dns.TypeToString[qtype])
	// IncrementMetricsQueryRequestsTotal(zname, qtype)

	key := c.BuildConsulKey(zname, rname)
	logging.Log.Debugf("Constructed Consul key '%s'", key)

	record, err := c.GetRecordFromConsul(key)
	if err != nil {
		logging.Log.Errorf("Error receiving key '%s' from consul: %v", key, err)
		IncrementMetricsPluginErrorsTotal("CONSUL_GET")

		return false
	}

	if record != nil {
		return c.HandleRecord(msg, rname, dns.TypeA, record)
	}

	logging.Log.Debugf("No record found for alias '%s' and type '%s'", alias, dns.TypeToString[qtype])

	return true
}
