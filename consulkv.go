package consulkv

import (
	"context"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
	"github.com/mwantia/coredns-consulkv-plugin/records"
)

func (c ConsulKV) Name() string { return "consulkv" }

func (c ConsulKV) ServeDNS(ctx context.Context, writer dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: writer, Req: r}
	qname := state.Name()
	qtype := state.QType()

	logging.Log.Debugf("Received query for %s", qname)

	zname, rname := c.GetZoneAndRecord(qname)
	if zname == "" {
		logging.Log.Debugf("Zone %s not in configured zones %s, passing to next plugin", zname, c.Zones)

		return plugin.NextOrFailure(c.Name(), c.Next, ctx, writer, r)
	}

	logging.Log.Debugf("Received new request for zone '%s' and record '%s' with code '%s", zname, rname, dns.TypeToString[qtype])
	IncrementMetricsQueryRequestsTotal(zname, qtype)

	key := c.BuildConsulKey(zname, rname)
	logging.Log.Debugf("Constructed Consul key '%s'", key)

	record, err := c.GetRecordFromConsul(key)
	if err != nil {
		logging.Log.Errorf("Error receiving key '%s' from consul: %v", key, err)
		IncrementMetricsPluginErrorsTotal("CONSUL_GET")

		return HandleConsulError(zname, r, writer, err)
	}

	if record == nil {
		return c.HandleMissingRecord(qname, qtype, zname, rname, ctx, writer, r)
	}

	return c.CreateDNSResponse(qname, qtype, record, ctx, r, writer)
}

func (c ConsulKV) HandleMissingRecord(qname string, qtype uint16, zname string, rname string, ctx context.Context, writer dns.ResponseWriter, r *dns.Msg) (int, error) {
	soa, err := c.GetSOARecordFromConsul(zname)
	if err != nil {
		logging.Log.Errorf("Error loading SOA record: %v", err)
		IncrementMetricsPluginErrorsTotal("SOA_GET")
		IncrementMetricsResponsesFailedTotal(zname, qtype, "ERROR")

		return c.HandleError(r, dns.RcodeNameError, writer, nil)
	}

	if rname == "@" {
		logging.Log.Warning("No root entry found in Consul")
		IncrementMetricsResponsesFailedTotal(zname, qtype, "NXDOMAIN")

		return c.HandleNXDomain(qname, soa, r, writer)
	}

	key := c.BuildConsulKey(zname, "*")

	record, err := c.GetRecordFromConsul(key)
	if err != nil {
		logging.Log.Errorf("Error receiving key '%s' from consul: %v", key, err)
		IncrementMetricsPluginErrorsTotal("CONSUL_GET")

		return HandleConsulError(zname, r, writer, err)
	}

	if record == nil {
		logging.Log.Warningf("No record found for zone '%s' and record '%s'", zname, rname)
		IncrementMetricsResponsesFailedTotal(zname, qtype, "NXDOMAIN")

		return c.HandleNXDomain(qname, soa, r, writer)
	}

	return c.CreateDNSResponse(qname, qtype, record, ctx, r, writer)
}

func (c ConsulKV) CreateDNSResponse(qname string, qtype uint16, record *records.Record, ctx context.Context, r *dns.Msg, writer dns.ResponseWriter) (int, error) {
	msg := PrepareResponseReply(r, false)

	logging.Log.Debugf("Creating DNS response for %s", qname)

	handled := c.HandleRecord(msg, qname, qtype, record)
	zname, _ := c.GetZoneAndRecord(qname)

	if handled && len(msg.Answer) > 0 {
		return c.SendDNSResponse(zname, msg, writer)
	}

	return c.HandleNoMatchingRecords(qname, qtype, ctx, r, writer)
}

func (c ConsulKV) SendDNSResponse(zname string, msg *dns.Msg, writer dns.ResponseWriter) (int, error) {
	logging.Log.Debugf("Sending DNS response with %d answers", len(msg.Answer))
	err := writer.WriteMsg(msg)

	if err != nil {
		logging.Log.Errorf("Error writing DNS response: %v", err)
		IncrementMetricsPluginErrorsTotal("WRITE_MSG")

		return c.HandleError(msg, dns.RcodeServerFailure, writer, err)
	}

	for _, q := range msg.Question {
		IncrementMetricsResponsesSuccessfulTotal(zname, q.Qtype)
	}

	return dns.RcodeSuccess, nil
}

func (c ConsulKV) HandleNoMatchingRecords(qname string, qtype uint16, ctx context.Context, request *dns.Msg, writer dns.ResponseWriter) (int, error) {
	zname, rname := c.GetZoneAndRecord(qname)

	logging.Log.Infof("No matching record was found for zone '%s' and record '%s' with code '%s'",
		zname, rname, dns.TypeToString[qtype])

	soa, err := c.GetSOARecordFromConsul(zname)
	if err != nil {
		logging.Log.Errorf("Error loading SOA record: %v", err)
		IncrementMetricsPluginErrorsTotal("SOA_GET")
		IncrementMetricsResponsesFailedTotal(zname, qtype, "ERROR")

		return c.HandleError(request, dns.RcodeNameError, writer, nil)
	}

	if soa != nil {
		IncrementMetricsResponsesFailedTotal(zname, qtype, "NODATA")
		return c.HandleNoData(qname, soa, request, writer)
	}

	IncrementMetricsResponsesFailedTotal(zname, qtype, "NXDOMAIN")
	return c.HandleNXDomain(qname, soa, request, writer)
}

func (c ConsulKV) HandleNXDomain(qname string, soa *records.SOARecord, request *dns.Msg, writer dns.ResponseWriter) (int, error) {
	m := PrepareResponseRcode(request, dns.RcodeNameError, false)

	records.AppendSOAToAuthority(m, qname, soa)

	err := writer.WriteMsg(m)
	if err != nil {
		logging.Log.Errorf("Error writing NODATA response: %v", err)
		IncrementMetricsPluginErrorsTotal("WRITE_MSG")

		return dns.RcodeServerFailure, err
	}

	return dns.RcodeNameError, nil
}

func (c ConsulKV) HandleNoData(qname string, soa *records.SOARecord, request *dns.Msg, writer dns.ResponseWriter) (int, error) {
	m := PrepareResponseReply(request, false)

	records.AppendSOAToAuthority(m, qname, soa)

	err := writer.WriteMsg(m)
	if err != nil {
		logging.Log.Errorf("Error writing NODATA response: %v", err)
		IncrementMetricsPluginErrorsTotal("WRITE_MSG")

		return dns.RcodeServerFailure, err
	}

	return dns.RcodeSuccess, nil
}

func (c ConsulKV) HandleError(request *dns.Msg, rcode int, writer dns.ResponseWriter, e error) (int, error) {
	m := PrepareResponseRcode(request, rcode, true)

	err := writer.WriteMsg(m)
	if err != nil {
		logging.Log.Errorf("Error writing DNS error response: %v", err)
		IncrementMetricsPluginErrorsTotal("WRITE_MSG")

		return dns.RcodeServerFailure, err
	}

	return rcode, e
}
