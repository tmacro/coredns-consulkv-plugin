package consulkv

import (
	"context"
	"encoding/json"
	"net"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/hashicorp/consul/api"
	"github.com/miekg/dns"
)

type ConsulKV struct {
	Next        plugin.Handler
	Client      *api.Client
	Prefix      string
	Address     string
	Token       string
	Zones       []string
	Fallthrough bool
}

func (c ConsulKV) Name() string { return pluginname }

func (c ConsulKV) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	qname := state.Name()
	qtype := state.QType()

	log.Debugf("Received query for %s", qname)

	// Remove the trailing dot if present
	qname = strings.TrimSuffix(dns.Fqdn(qname), ".")

	// Find the matching zone
	var zoneName string
	var recordName string
	for _, zone := range c.Zones {
		if strings.HasSuffix(dns.Fqdn(qname), dns.Fqdn(zone)) {
			zoneName = zone
			recordName = qname

			for strings.HasSuffix(recordName, zone) {
				recordName = strings.TrimSuffix(recordName, zone)
				recordName = strings.TrimSuffix(recordName, ".")
			}

			break
		}
	}

	if zoneName == "" {
		log.Debugf("Zone %s not in configured zones %s, passing to next plugin", zoneName, c.Zones)
		return plugin.NextOrFailure(c.Name(), c.Next, ctx, w, r)
	}

	if recordName == "" {
		recordName = "@"
	}

	log.Debugf("Record: %s, Zone: %s", recordName, zoneName)

	key := c.Prefix + "/" + zoneName + "/" + recordName
	log.Debugf("Constructed key: %s", key)

	kv, _, err := c.Client.KV().Get(key, nil)
	if err != nil {
		log.Errorf("Error fetching from Consul: %v", err)

		consulErrors.WithLabelValues(dns.Fqdn(zoneName)).Inc()
		return c.HandleError(r, dns.RcodeServerFailure, w, err)
	}
	if kv == nil {
		if recordName == "@" {
			if c.Fallthrough {
				return plugin.NextOrFailure(c.Name(), c.Next, ctx, w, r)
			}

			failedQueries.WithLabelValues(dns.Fqdn(zoneName)).Inc()
			log.Warning("No root entry found in Consul")
			return c.HandleError(r, dns.RcodeNameError, w, nil)
		}
		//
		wildcardkey := c.Prefix + "/" + zoneName + "/*"
		wildcardkv, _, err := c.Client.KV().Get(wildcardkey, nil)

		if err != nil {
			log.Errorf("Error fetching from Consul: %v", err)

			consulErrors.WithLabelValues(dns.Fqdn(zoneName)).Inc()
			return c.HandleError(r, dns.RcodeServerFailure, w, err)
		}

		if wildcardkv == nil {
			if c.Fallthrough {
				return plugin.NextOrFailure(c.Name(), c.Next, ctx, w, r)
			}

			log.Warningf("No value found in Consul for key: %s", key)
			failedQueries.WithLabelValues(dns.Fqdn(zoneName)).Inc()
			return c.HandleError(r, dns.RcodeNameError, w, nil)
		}

		kv = wildcardkv
	}

	log.Debugf("Found value in Consul: %s", string(kv.Value))

	var record struct {
		TTL     *int `json:"ttl"`
		Records []struct {
			Type  string          `json:"type"`
			Value json.RawMessage `json:"value"`
		} `json:"records"`
	}

	err = json.Unmarshal(kv.Value, &record)
	if err != nil {
		log.Errorf("Error parsing JSON: %v", err)
		invalidResponses.WithLabelValues(dns.Fqdn(zoneName)).Inc()

		return c.HandleError(r, dns.RcodeServerFailure, w, err)
	}

	ttl := 3600
	if record.TTL != nil {
		ttl = *record.TTL
	}

	a := new(dns.Msg)
	a.SetReply(r)
	a.Authoritative = true

	foundRequestedType := false

	for _, rec := range record.Records {
		switch rec.Type {
		case "A":
			var values []string
			if err := json.Unmarshal(rec.Value, &values); err != nil {
				log.Errorf("Error parsing JSON for A record: %v", err)
				invalidResponses.WithLabelValues(dns.Fqdn(zoneName)).Inc()

				return c.HandleError(r, dns.RcodeServerFailure, w, err)
			}

			if qtype == dns.TypeA {
				if qtype == dns.TypeA {
					for _, ip := range values {
						rr := &dns.A{
							Hdr: dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: uint32(ttl)},
							A:   net.ParseIP(ip),
						}
						a.Answer = append(a.Answer, rr)
						foundRequestedType = true
					}
				}
			}

		case "AAAA":
			var values []string
			if err := json.Unmarshal(rec.Value, &values); err != nil {
				log.Errorf("Error parsing JSON for AAAA record: %v", err)
				invalidResponses.WithLabelValues(dns.Fqdn(zoneName)).Inc()

				return c.HandleError(r, dns.RcodeServerFailure, w, err)
			}

			if qtype == dns.TypeAAAA {
				for _, ip := range values {
					rr := &dns.AAAA{
						Hdr:  dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: uint32(ttl)},
						AAAA: net.ParseIP(ip),
					}
					a.Answer = append(a.Answer, rr)
					foundRequestedType = true
				}
			}

		case "CNAME":
			var value string
			if err := json.Unmarshal(rec.Value, &value); err != nil {
				log.Errorf("Error parsing JSON for CNAME record: %v", err)
				invalidResponses.WithLabelValues(dns.Fqdn(zoneName)).Inc()

				return c.HandleError(r, dns.RcodeServerFailure, w, err)
			}

			if qtype == dns.TypeCNAME && value != "" {
				rr := &dns.CNAME{
					Hdr:    dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: uint32(ttl)},
					Target: dns.Fqdn(value),
				}
				a.Answer = append(a.Answer, rr)
			}

		case "PTR":
			var values []string
			if err := json.Unmarshal(rec.Value, &values); err != nil {
				log.Errorf("Error parsing JSON for PTR record: %v", err)
				invalidResponses.WithLabelValues(dns.Fqdn(zoneName)).Inc()

				return c.HandleError(r, dns.RcodeServerFailure, w, err)
			}

			if qtype == dns.TypePTR {
				for _, ptr := range values {
					rr := &dns.PTR{
						Hdr: dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: uint32(ttl)},
						Ptr: dns.Fqdn(ptr),
					}
					a.Answer = append(a.Answer, rr)
				}
			}

		case "SRV":
			var srvValues []struct {
				Target   string `json:"target"`
				Port     uint16 `json:"port"`
				Priority uint16 `json:"priority"`
				Weight   uint16 `json:"weight"`
			}

			if err := json.Unmarshal(rec.Value, &srvValues); err != nil {
				log.Errorf("Error parsing JSON for SRV record: %v", err)
				invalidResponses.WithLabelValues(dns.Fqdn(zoneName)).Inc()

				return c.HandleError(r, dns.RcodeServerFailure, w, err)
			}

			if qtype == dns.TypeSRV {
				for _, srv := range srvValues {
					rr := &dns.SRV{
						Hdr:      dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: uint32(ttl)},
						Priority: srv.Priority,
						Weight:   srv.Weight,
						Port:     srv.Port,
						Target:   dns.Fqdn(srv.Target),
					}
					a.Answer = append(a.Answer, rr)
				}
			}

		case "TXT":
			var values []string
			if err := json.Unmarshal(rec.Value, &values); err != nil {
				log.Errorf("Error parsing JSON for TXT record: %v", err)
				return dns.RcodeServerFailure, err
			}

			rr := &dns.TXT{
				Hdr: dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: uint32(ttl)},
				Txt: values,
			}

			if qtype == dns.TypeTXT {
				a.Answer = append(a.Answer, rr)
				foundRequestedType = true

			} else if qtype == dns.TypeA {
				a.Extra = append(a.Extra, rr)
			}
		}
	}

	if foundRequestedType && len(a.Answer) > 0 {
		log.Debugf("Sending DNS response with %d answers", len(a.Answer))
		err := w.WriteMsg(a)
		if err != nil {
			log.Errorf("Error writing DNS response: %v", err)
			invalidResponses.WithLabelValues(dns.Fqdn(zoneName)).Inc()

			return c.HandleError(r, dns.RcodeServerFailure, w, err)
		}

		successfulQueries.WithLabelValues(dns.Fqdn(zoneName)).Inc()
		return dns.RcodeSuccess, nil
	}

	if c.Fallthrough {
		return plugin.NextOrFailure(c.Name(), c.Next, ctx, w, r)
	}

	log.Warningf("Requested record type %d not found for %s", qtype, qname)
	failedQueries.WithLabelValues(dns.Fqdn(zoneName)).Inc()
	return c.HandleError(r, dns.RcodeNameError, w, nil)
}

func (c ConsulKV) HandleError(r *dns.Msg, rcode int, w dns.ResponseWriter, e error) (int, error) {
	m := new(dns.Msg)
	m.SetRcode(r, rcode)
	m.Authoritative = true
	m.RecursionAvailable = true

	err := w.WriteMsg(m)
	if err != nil {
		log.Errorf("Error writing DNS error response: %v", err)
		return dns.RcodeServerFailure, err
	}

	return rcode, e
}
