package consulkv

import (
	"encoding/json"

	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
)

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

	if c.NoFlattening {
		logging.Log.Debugf("CNAME flattening disabled; Only returning CNAME record for '%s'", alias)

		return true
	}

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
