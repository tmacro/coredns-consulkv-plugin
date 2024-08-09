package records

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"

	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
)

type SVCBRecord struct {
	Priority uint16            `json:"priority"`
	Target   string            `json:"target"`
	Params   map[string]string `json:"params"`
}

type HTTPSRecord = SVCBRecord

func AppendSVCBRecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage, recordType uint16) (bool, error) {
	var svcbs []SVCBRecord
	if err := json.Unmarshal(value, &svcbs); err != nil {
		return false, err
	}

	for _, svcb := range svcbs {
		rr := &dns.SVCB{
			Hdr:      dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: recordType, Class: dns.ClassINET, Ttl: uint32(ttl)},
			Priority: svcb.Priority,
			Target:   dns.Fqdn(svcb.Target),
		}

		for key, value := range svcb.Params {
			svcbKey := SVCBKeyToCode(key)
			var svcbValue dns.SVCBKeyValue

			switch svcbKey {
			case dns.SVCB_ALPN:
				svcbValue = &dns.SVCBAlpn{Alpn: strings.Split(value, ",")}
			case dns.SVCB_NO_DEFAULT_ALPN:
				svcbValue = &dns.SVCBNoDefaultAlpn{}
			case dns.SVCB_PORT:
				port, _ := strconv.Atoi(value)
				svcbValue = &dns.SVCBPort{Port: uint16(port)}
			case dns.SVCB_IPV4HINT:
				ips := strings.Split(value, ",")
				ipv4 := make([]net.IP, len(ips))
				for i, ip := range ips {
					ipv4[i] = net.ParseIP(ip)
				}
				svcbValue = &dns.SVCBIPv4Hint{Hint: ipv4}
			case dns.SVCB_IPV6HINT:
				ips := strings.Split(value, ",")
				ipv6 := make([]net.IP, len(ips))
				for i, ip := range ips {
					ipv6[i] = net.ParseIP(ip)
				}
				svcbValue = &dns.SVCBIPv6Hint{Hint: ipv6}
			case dns.SVCB_DOHPATH:
				svcbValue = &dns.SVCBDoHPath{Template: value}
			default:
				svcbValue = &dns.SVCBLocal{KeyCode: svcbKey, Data: []byte(value)}
			}

			rr.Value = append(rr.Value, svcbValue)
		}

		msg.Answer = append(msg.Answer, rr)
	}

	return len(svcbs) > 0, nil
}

func SVCBKeyToCode(key string) dns.SVCBKey {
	switch strings.ToLower(key) {
	case "mandatory":
		return dns.SVCB_MANDATORY
	case "alpn":
		return dns.SVCB_ALPN
	case "no-default-alpn":
		return dns.SVCB_NO_DEFAULT_ALPN
	case "port":
		return dns.SVCB_PORT
	case "ipv4hint":
		return dns.SVCB_IPV4HINT
	case "ipv6hint":
		return dns.SVCB_IPV6HINT
	case "dohpath":
		return dns.SVCB_DOHPATH

	default:
		code, err := strconv.Atoi(key)
		if err == nil && code >= 0 && code <= 65535 {
			return dns.SVCBKey(code)
		}

		logging.Log.Warningf("Unknown SVCB key: %s", key)

		return dns.SVCBKey(0) // Use 0 for unknown keys
	}
}
