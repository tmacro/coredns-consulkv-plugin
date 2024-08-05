package consulkv

import (
	"encoding/json"
	"net"

	"github.com/miekg/dns"
)

func AppendARecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) bool {
	var ips []string
	if err := json.Unmarshal(value, &ips); err != nil {
		log.Errorf("Error parsing JSON for A record: %v", err)
		return false
	}

	for _, ip := range ips {
		rr := &dns.A{
			Hdr: dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: uint32(ttl)},
			A:   net.ParseIP(ip),
		}
		msg.Answer = append(msg.Answer, rr)
	}

	return len(ips) > 0
}

func AppendAAAARecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) bool {
	var ips []string
	if err := json.Unmarshal(value, &ips); err != nil {
		log.Errorf("Error parsing JSON for AAAA record: %v", err)
		return false
	}

	for _, ip := range ips {
		rr := &dns.AAAA{
			Hdr:  dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: uint32(ttl)},
			AAAA: net.ParseIP(ip),
		}
		msg.Answer = append(msg.Answer, rr)
	}

	return len(ips) > 0
}

func (c *ConsulKV) AppendCNAMERecords(msg *dns.Msg, qname string, qtype uint16, ttl int, value json.RawMessage) bool {
	var alias string
	if err := json.Unmarshal(value, &alias); err != nil {
		log.Errorf("Error parsing JSON for CNAME record: %v", err)
		return false
	}

	rr := &dns.CNAME{
		Hdr:    dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: uint32(ttl)},
		Target: dns.Fqdn(alias),
	}
	msg.Answer = append(msg.Answer, rr)

	zoneName, recordName := c.GetZoneAndRecordName(alias)
	if zoneName == "" {
		log.Debugf("Zone %s not in configured zones %s, passing to next plugin", zoneName, c.Zones)
	}

	log.Debugf("Record: %s, Zone: %s", recordName, zoneName)

	key := BuildConsulKey(c.Prefix, zoneName, recordName)

	log.Debugf("Constructed key: %s", key)

	record, err := c.GetRecordFromConsul(key)
	if err != nil {
		log.Errorf("Error resolving CNAME alias %s: %v", recordName, err)
		return true
	}

	if record != nil {
		c.HandleRecord(msg, recordName, qtype, record)
	}

	log.Debugf("No record found for alias %s and type %v", recordName, qtype)

	return true
}

func AppendPTRRecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) bool {
	var ips []string
	if err := json.Unmarshal(value, &ips); err != nil {
		log.Errorf("Error parsing JSON for PTR record: %v", err)
		return false
	}

	for _, ptr := range ips {
		rr := &dns.PTR{
			Hdr: dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: uint32(ttl)},
			Ptr: dns.Fqdn(ptr),
		}
		msg.Answer = append(msg.Answer, rr)
	}

	return len(ips) > 0
}

func AppendSRVRecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) bool {
	var records []SRVRecord
	if err := json.Unmarshal(value, &records); err != nil {
		log.Errorf("Error parsing JSON for SRV record: %v", err)
		return false
	}

	for _, record := range records {
		rr := &dns.SRV{
			Hdr:      dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: uint32(ttl)},
			Priority: record.Priority,
			Weight:   record.Weight,
			Port:     record.Port,
			Target:   dns.Fqdn(record.Target),
		}
		msg.Answer = append(msg.Answer, rr)
	}

	return len(records) > 0
}

func AppendTXTRecords(msg *dns.Msg, qtype uint16, qname string, ttl int, value json.RawMessage) bool {
	var values []string
	if err := json.Unmarshal(value, &values); err != nil {
		log.Errorf("Error parsing JSON for TXT record: %v", err)
		return false
	}

	txtAnswered := false

	rr := &dns.TXT{
		Hdr: dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: uint32(ttl)},
		Txt: values,
	}

	if qtype == dns.TypeTXT {
		msg.Answer = append(msg.Answer, rr)
		txtAnswered = true
	} else if qtype == dns.TypeA {
		msg.Extra = append(msg.Extra, rr)
	}

	return txtAnswered
}
