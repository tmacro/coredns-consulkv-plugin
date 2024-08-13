package consulkv

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/mwantia/coredns-consulkv-plugin/logging"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
)

type TestCase struct {
	testName       string
	queryName      string
	queryType      uint16
	expectedCode   int
	expectedType   uint16
	expectedAnswer string
}

func TestConsulKV(tst *testing.T) {
	OverwriteStdOut()
	clog.D.Set()

	c := caddy.NewTestController("dns", `
		consulkv {
		  token 93804aef-7648-c6ce-e609-4f75676bc6c3
		}
	`)

	plug := &ConsulKVPlugin{}

	consul, err := CreateConsulConfig(c)
	if err != nil {
		tst.Errorf("Unable to create consul client: %v", err)
	}

	config, err := consul.GetConfigFromConsul()
	if err != nil {
		tst.Errorf("Unable to get config from consul: %v", err)
	}

	plug.Consul = consul
	plug.Config = config

	tests := GenerateTestCases()
	RunTests(tst, plug, tests)
}

func OverwriteStdOut() error {
	tempFile, err := os.CreateTemp("", "coredns-consulkv-test-log")
	if err != nil {
		return err
	}

	defer os.Remove(tempFile.Name())

	orig := logging.Log
	logging.Log = clog.NewWithPlugin("consulkv")
	log.SetOutput(os.Stdout)

	defer func() {
		logging.Log = orig
	}()

	return nil
}

func GenerateTestCases() []TestCase {
	return []TestCase{
		{"NS", "example.com", dns.TypeNS, dns.RcodeSuccess, dns.TypeNS, "ns.example.com"},
		{"@", "example.com", dns.TypeA, dns.RcodeSuccess, dns.TypeA, "192.168.0.2"},
		{"A", "www.example.com", dns.TypeA, dns.RcodeSuccess, dns.TypeA, "192.168.0.3"},
		{"CNAME", "alias.example.com", dns.TypeCNAME, dns.RcodeSuccess, dns.TypeCNAME, "www.example.com"},
		{"TXT", "txt.example.com", dns.TypeTXT, dns.RcodeSuccess, dns.TypeTXT, "This is a test"},
	}
}

func RunTests(tst *testing.T, plug *ConsulKVPlugin, tests []TestCase) {
	ctx := context.TODO()

	for _, tc := range tests {
		tst.Run(tc.testName, func(t *testing.T) {

			logging.Log.Debugf("Testing %s query for %s with type %s",
				tc.testName, tc.queryName, dns.TypeToString[tc.queryType])

			req := new(dns.Msg)
			req.SetQuestion(tc.queryName, tc.queryType)
			rec := dnstest.NewRecorder(&test.ResponseWriter{})

			code, err := plug.ServeDNS(ctx, rec, req)

			if err != nil {
				tst.Errorf("Expected no error, but got: %v", err)
			}

			if code != tc.expectedCode {
				tst.Errorf("Expected rcode %d, but got %d", tc.expectedCode, code)
			}

			if tc.expectedAnswer != "" {
				if len(rec.Msg.Answer) == 0 {
					tst.Errorf("Expected an answer, but got none")
				} else {
					answer := rec.Msg.Answer[0]
					switch tc.queryType {
					case dns.TypeA:
						if a, ok := answer.(*dns.A); ok {
							if a.A.String() != tc.expectedAnswer {
								tst.Errorf("Expected IP %s, but got %s", tc.expectedAnswer, a.A.String())
							}
						} else {
							tst.Errorf("Expected A record, but got %T", answer)
						}
					case dns.TypeCNAME:
						if cname, ok := answer.(*dns.CNAME); ok {
							if dns.Fqdn(cname.Target) != dns.Fqdn(tc.expectedAnswer) {
								tst.Errorf("Expected CNAME %s, but got %s", tc.expectedAnswer, cname.Target)
							}
						} else {
							tst.Errorf("Expected CNAME record, but got %T", answer)
						}
					case dns.TypeTXT:
						if txt, ok := answer.(*dns.TXT); ok {
							if txt.Txt[0] != tc.expectedAnswer {
								tst.Errorf("Expected TXT %s, but got %s", tc.expectedAnswer, txt.Txt[0])
							}
						} else {
							tst.Errorf("Expected TXT record, but got %T", answer)
						}
					}
				}
			}
		})
	}
}
