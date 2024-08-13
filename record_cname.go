package consulkv

import (
	"context"
	"encoding/json"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
	"github.com/mwantia/coredns-consulkv-plugin/types"
)

func (plug *ConsulKVPlugin) AppendCNAMERecords(ctx context.Context, msg *dns.Msg, qname string, qtype uint16, ttl int, value json.RawMessage) bool {
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

	if plug.Config.Flattening == types.Flattening_None {
		logging.Log.Debugf("CNAME flattening disabled; Only returning CNAME record for '%s'", alias)

		return true
	}

	zname, rname := GetZoneAndRecord(plug.Config.Zones, alias)
	if zname == "" {
		if plug.Config.Flattening == types.Flattening_Full {
			logging.Log.Debugf("Zone %s not in configured zones %s, passing to next plugin ", zname, plug.Config.Zones)
			return plug.HandleExternalCNAME(ctx, msg, alias, qtype)
		}

		logging.Log.Debugf("Zone %s not in configured zones %s, skipping CNAME flattening", zname, plug.Config.Zones)
		return true
	}

	logging.Log.Debugf("Received new request for zone '%s' and record '%s' with code '%s", zname, rname, dns.TypeToString[qtype])

	record, err := plug.Consul.GetZoneRecordFromConsul(zname, rname, plug.Config.ConsulCache)
	if err != nil {
		logging.Log.Errorf("Error receiving value for zone '%s' and name '%s': %v", zname, rname, err)
		IncrementMetricsPluginErrorsTotal("CONSUL_GET")

		return false
	}

	if record != nil {
		return plug.HandleRecord(ctx, msg, rname, dns.TypeA, record)
	}

	logging.Log.Debugf("No record found for alias '%s' and type '%s'", alias, dns.TypeToString[qtype])

	return true
}

func (plug *ConsulKVPlugin) HandleExternalCNAME(ctx context.Context, msg *dns.Msg, alias string, qtype uint16) bool {
	logging.Log.Debugf("Resolving external CNAME target: %s", alias)

	request := request.Request{W: &ResponseWriterWrapper{WrappedMsg: msg}, Req: new(dns.Msg)}
	request.Req.SetQuestion(dns.Fqdn(alias), qtype)

	_, err := plugin.NextOrFailure(plug.Name(), plug.Next, ctx, request.W, request.Req)
	if err != nil {
		logging.Log.Errorf("Error in external resolution: %v", err)
		return false
	}

	return len(msg.Answer) > len(msg.Answer)-1
}
