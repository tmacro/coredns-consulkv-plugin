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

	key := BuildConsulKey(c.Prefix, zoneName, recordName)

	logging.Log.Debugf("Constructed key: %s", key)

	record, err := c.GetRecordFromConsul(key)
	if err != nil {
		return c.HandleConsulError(zoneName, r, w, err)
	}

	if record == nil {
		return c.HandleMissingRecord(qname, qtype, zoneName, recordName, ctx, w, r)
	}

	return c.CreateDNSResponse(qname, qtype, record, ctx, r, writer)
}

func (c ConsulKV) HandleMissingRecord(qname string, qtype uint16, zoneName string, recordName string,
	ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	if recordName == "@" {
		failedQueries.WithLabelValues(dns.Fqdn(zoneName)).Inc()
		logging.Log.Warning("No root entry found in Consul")
		IncrementMetricsResponsesFailedTotal(zname, qtype, "NXDOMAIN")

		return c.HandleError(r, dns.RcodeNameError, w, nil)
	}

	key := BuildConsulKey(c.Prefix, zoneName, "*")
	record, err := c.GetRecordFromConsul(key)
	if err != nil {
		return c.HandleConsulError(zoneName, r, w, err)
	}

	if record == nil {
		logging.Log.Warningf("No value found in Consul for key: %s", key)
		failedQueries.WithLabelValues(dns.Fqdn(zoneName)).Inc()

		soa, err := c.GetSOARecordFromConsul(zoneName)
		if err != nil {
			logging.Log.Errorf("Error loading SOA record: %v", err)

			return c.HandleError(r, dns.RcodeNameError, w, nil)
		}

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
		return c.SendDNSResponse(zoneName, msg, w)
	}

	return c.HandleNoMatchingRecords(qname, qtype, ctx, r, writer)
}

func (c ConsulKV) SendDNSResponse(zoneName string, msg *dns.Msg, w dns.ResponseWriter) (int, error) {
	logging.Log.Debugf("Sending DNS response with %d answers", len(msg.Answer))
	err := writer.WriteMsg(msg)

	if err != nil {
		logging.Log.Errorf("Error writing DNS response: %v", err)
		IncrementMetricsPluginErrorsTotal("WRITE_MSG")

		return c.HandleError(msg, dns.RcodeServerFailure, writer, err)
	}

		return c.HandleError(msg, dns.RcodeServerFailure, w, err)
	}

	return dns.RcodeSuccess, nil
}

func (c ConsulKV) HandleNoMatchingRecords(qname string, qtype uint16, ctx context.Context, r *dns.Msg, w dns.ResponseWriter) (int, error) {
	logging.Log.Infof("Requested record type %s not found for %s", dns.TypeToString[qtype], qname)
	failedQueries.WithLabelValues(dns.Fqdn(qname)).Inc()

	zoneName, _ := c.GetZoneAndRecordName(qname)
	soa, err := c.GetSOARecordFromConsul(zoneName)

	if err == nil && soa != nil {
		return c.HandleNoData(qname, soa, r, w)
	}

	return c.HandleNXDomain(qname, soa, r, w)
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
