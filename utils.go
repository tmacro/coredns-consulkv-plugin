package consulkv

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
	"github.com/mwantia/coredns-consulkv-plugin/records"
)

func LoadEnvFile(path string) error {
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		path = filepath.Join(cwd, path)
	}

	err := godotenv.Load(path)
	if err != nil {
		logging.Log.Warningf("Unable to load environment file '%s'; Ignore if not required: %v", path, err)
		return err
	}

	logging.Log.Infof("Loaded environment file '%s'", path)
	return nil
}

func GetEnvOrDefault(key, defaultvalue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultvalue
	}
	return value
}

func PrepareResponseRcode(request *dns.Msg, rcode int, recursionAvailable bool) *dns.Msg {
	m := new(dns.Msg)
	m.SetRcode(request, rcode)
	m.Authoritative = true
	m.RecursionAvailable = recursionAvailable

	return m
}

func PrepareResponseReply(request *dns.Msg, recursionAvailable bool) *dns.Msg {
	m := new(dns.Msg)
	m.SetReply(request)
	m.Authoritative = true
	m.RecursionAvailable = recursionAvailable

	return m
}

func GetZoneAndRecord(zones []string, qname string) (string, string) {
	qname = strings.TrimSuffix(dns.Fqdn(qname), ".")

	for _, zone := range zones {
		if strings.HasSuffix(qname, zone) {
			record := strings.TrimSuffix(qname, zone)
			record = strings.TrimSuffix(record, ".")

			if record == "" {
				record = "@"
			}

			return zone, record
		}
	}

	return "", ""
}

func GetDefaultSOA(zoneName string) *records.SOARecord {
	return &records.SOARecord{
		MNAME:   "ns." + zoneName,
		RNAME:   "hostmaster." + zoneName,
		SERIAL:  soaSerial,
		REFRESH: 3600,
		RETRY:   600,
		EXPIRE:  86400,
		MINIMUM: 3600,
	}
}

func GetDefaultTTL(record *records.Record) int {
	if record.TTL != nil {
		return *record.TTL
	}

	return 3600 // Default TTL
}
