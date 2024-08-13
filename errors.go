package consulkv

import (
	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
	"github.com/mwantia/coredns-consulkv-plugin/records"
)

func HandleError(request *dns.Msg, rcode int, writer dns.ResponseWriter, e error) (int, error) {
	m := PrepareResponseRcode(request, rcode, true)

	err := writer.WriteMsg(m)
	if err != nil {
		logging.Log.Errorf("Error writing DNS error response: %v", err)
		IncrementMetricsPluginErrorsTotal("WRITE_MSG")

		return dns.RcodeServerFailure, err
	}

	return rcode, e
}

func HandleNXDomain(qname string, soa *records.SOARecord, request *dns.Msg, writer dns.ResponseWriter) (int, error) {
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

func HandleNoData(qname string, soa *records.SOARecord, request *dns.Msg, writer dns.ResponseWriter) (int, error) {
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

func HandleConsulError(r *dns.Msg, w dns.ResponseWriter, e error) (int, error) {
	m := new(dns.Msg)
	m.SetRcode(r, dns.RcodeServerFailure)
	m.Authoritative = true
	m.RecursionAvailable = true

	err := w.WriteMsg(m)
	if err != nil {
		logging.Log.Errorf("Unable to write DNS error response: %v", err)
		return dns.RcodeServerFailure, err
	}

	return dns.RcodeServerFailure, e
}
