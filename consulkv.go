package consulkv

import (
	"context"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
	"github.com/mwantia/coredns-consulkv-plugin/records"
)

func (plug ConsulKVPlugin) Name() string { return "consulkv" }

func (plug ConsulKVPlugin) ServeDNS(ctx context.Context, writer dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: writer, Req: r}
	qname := state.Name()
	qtype := state.QType()

	logging.Log.Debugf("Received query for %s", qname)

	zname, rname := GetZoneAndRecord(plug.Config.Zones, qname)
	if zname == "" {
		logging.Log.Debugf("Name %s not in configured zones %s, passing to next plugin", qname, plug.Config.Zones)

		return plugin.NextOrFailure(plug.Name(), plug.Next, ctx, writer, r)
	}

	logging.Log.Debugf("Received new request for zone '%s' and record '%s' with code '%s", zname, rname, dns.TypeToString[qtype])
	IncrementMetricsQueryRequestsTotal(zname, qtype)

	record, err := plug.Consul.GetZoneRecordFromConsul(zname, rname, plug.Config.ConsulCache)
	if err != nil {
		logging.Log.Errorf("Error receiving value for zone '%s' and name '%s': %v", zname, rname, err)
		IncrementMetricsPluginErrorsTotal("CONSUL_GET")
		IncrementMetricsResponsesFailedTotal(zname, qtype, "ERROR")

		return HandleConsulError(r, writer, err)
	}

	if record == nil {
		return plug.HandleMissingRecord(qname, qtype, zname, rname, ctx, writer, r)
	}

	return plug.CreateDNSResponse(qname, qtype, record, ctx, r, writer)
}

func (plug ConsulKVPlugin) HandleMissingRecord(qname string, qtype uint16, zname string, rname string, ctx context.Context, writer dns.ResponseWriter, r *dns.Msg) (int, error) {
	soa, err := plug.Consul.GetSOARecordFromConsul(zname, plug.Config.ConsulCache)
	if err != nil {
		logging.Log.Errorf("Error loading SOA record: %v", err)

		IncrementMetricsPluginErrorsTotal("SOA_GET")
		IncrementMetricsResponsesFailedTotal(zname, qtype, "ERROR")
		return HandleError(r, dns.RcodeNameError, writer, nil)
	}

	if rname == "@" {
		logging.Log.Warning("No root entry found in Consul")
		IncrementMetricsResponsesFailedTotal(zname, qtype, "NXDOMAIN")

		return HandleNXDomain(qname, soa, r, writer)
	}

	record, err := plug.Consul.GetZoneRecordFromConsul(zname, "*", plug.Config.ConsulCache)
	if err != nil {
		logging.Log.Errorf("Error receiving value for zone '%s' and name '*': %v", zname, err)

		IncrementMetricsPluginErrorsTotal("CONSUL_GET")
		IncrementMetricsResponsesFailedTotal(zname, qtype, "ERROR")
		return HandleConsulError(r, writer, err)
	}

	if record == nil {
		logging.Log.Warningf("No record found for zone '%s' and record '%s'", zname, rname)
		IncrementMetricsResponsesFailedTotal(zname, qtype, "NXDOMAIN")

		return HandleNXDomain(qname, soa, r, writer)
	}

	return plug.CreateDNSResponse(qname, qtype, record, ctx, r, writer)
}

func (plug ConsulKVPlugin) CreateDNSResponse(qname string, qtype uint16, record *records.Record, ctx context.Context, r *dns.Msg, writer dns.ResponseWriter) (int, error) {
	msg := PrepareResponseReply(r, false)

	logging.Log.Debugf("Creating DNS response for %s", qname)

	handled := plug.HandleRecord(ctx, msg, qname, qtype, record)
	zname, _ := GetZoneAndRecord(plug.Config.Zones, qname)

	if handled && len(msg.Answer) > 0 {
		return SendDNSResponse(zname, qtype, msg, writer)
	}

	return plug.HandleNoMatchingRecords(qname, qtype, ctx, r, writer)
}

func SendDNSResponse(zname string, qtype uint16, msg *dns.Msg, writer dns.ResponseWriter) (int, error) {
	logging.Log.Debugf("Sending DNS response with %d answers", len(msg.Answer))
	err := writer.WriteMsg(msg)

	if err != nil {
		logging.Log.Errorf("Error writing DNS response: %v", err)
		IncrementMetricsPluginErrorsTotal("WRITE_MSG")
		IncrementMetricsResponsesFailedTotal(zname, qtype, "ERROR")

		return HandleError(msg, dns.RcodeServerFailure, writer, err)
	}

	for _, q := range msg.Question {
		IncrementMetricsResponsesSuccessfulTotal(zname, q.Qtype)
	}

	return dns.RcodeSuccess, nil
}

func (plug ConsulKVPlugin) HandleNoMatchingRecords(qname string, qtype uint16, ctx context.Context, request *dns.Msg, writer dns.ResponseWriter) (int, error) {
	zname, rname := GetZoneAndRecord(plug.Config.Zones, qname)

	logging.Log.Infof("No matching record was found for zone '%s' and record '%s' with code '%s'",
		zname, rname, dns.TypeToString[qtype])

	soa, err := plug.Consul.GetSOARecordFromConsul(zname, plug.Config.ConsulCache)
	if err != nil {
		logging.Log.Errorf("Error loading SOA record: %v", err)

		IncrementMetricsPluginErrorsTotal("SOA_GET")
		IncrementMetricsResponsesFailedTotal(zname, qtype, "ERROR")
		return HandleError(request, dns.RcodeNameError, writer, nil)
	}

	if soa != nil {
		IncrementMetricsResponsesFailedTotal(zname, qtype, "NODATA")
		return HandleNoData(qname, soa, request, writer)
	}

	IncrementMetricsResponsesFailedTotal(zname, qtype, "NXDOMAIN")
	return HandleNXDomain(qname, soa, request, writer)
}
