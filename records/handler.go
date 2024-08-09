package records

import (
	"encoding/json"

	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
)

type Record struct {
	TTL     *int `json:"ttl"`
	Records []struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	} `json:"records"`
}

func HandleRecord(msg *dns.Msg, qname string, qtype uint16, record *Record) bool {
	ttl := GetRecordTTL(record)
	foundRequestedType := false

	logging.Log.Debugf("TTL: %v", ttl)

	return foundRequestedType
}

func GetRecordTTL(record *Record) int {
	if record.TTL != nil {
		return *record.TTL
	}

	return 3600 // Default TTL
}
