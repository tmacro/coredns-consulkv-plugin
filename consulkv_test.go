package consulkv

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/mwantia/coredns-consulkv-plugin/logging"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/test"
	"github.com/hashicorp/consul/api"
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

func TestConsulKV(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Errorf("Error loading .env file: %v", err)
	}

	tempFile, err := ioutil.TempFile("", "coredns-consulkv-test-log")
	if err != nil {
		t.Fatalf("Failed to create temp file for logging: %v", err)
	}
	defer os.Remove(tempFile.Name())

	originalLog := logging.Log
	logging.Log = clog.NewWithPlugin("consulkv")
	clog.D.Set()
	log.SetOutput(os.Stdout)
	defer func() {
		logging.Log = originalLog
	}()

	config := api.DefaultConfig()
	config.Address = os.Getenv("CONSUL_ADDRESS")
	if config.Address == "" {
		config.Address = "http://localhost:8500" // Default value if not set
	}
	config.Token = os.Getenv("CONSUL_TOKEN")

	client, err := api.NewClient(config)
	if err != nil {
		t.Errorf("Unable to create Consul client: %v", err)
	}

	c := ConsulKV{
		Client: client,
		Prefix: os.Getenv("CONSUL_PREFIX"),
		Zones:  strings.Split(os.Getenv("CONSUL_ZONES"), ","),
	}

	if c.Prefix == "" {
		c.Prefix = "dns"
	}
	if len(c.Zones) == 0 {
		c.Zones = []string{"example.com."}
	}

	var tests = []TestCase{
		{"NS", "example.com", dns.TypeNS, dns.RcodeSuccess, dns.TypeNS, ""},
		{"Root A", "example.com", dns.TypeA, dns.RcodeSuccess, dns.TypeA, "192.168.0.2"},
		{"Sub A", "www.example.com", dns.TypeA, dns.RcodeSuccess, dns.TypeA, "192.168.0.3"},
		{"CNAME Alias", "alias.example.com", dns.TypeCNAME, dns.RcodeSuccess, dns.TypeCNAME, "www.example.com"},
		{"TXT", "txt.example.com", dns.TypeTXT, dns.RcodeSuccess, dns.TypeTXT, "This is a test"},
	}

	runTests(t, &c, tests)
}

func runTests(t *testing.T, c *ConsulKV, tests []TestCase) {
	ctx := context.TODO()

	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			logging.Log.Debugf("Testing %s query for %s with type %s", tc.testName, tc.queryName, dns.TypeToString[tc.queryType])

			req := new(dns.Msg)
			req.SetQuestion(tc.queryName, tc.queryType)
			rec := dnstest.NewRecorder(&test.ResponseWriter{})

			code, err := c.ServeDNS(ctx, rec, req)

			if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}

			if code != tc.expectedCode {
				t.Errorf("Expected rcode %d, but got %d", tc.expectedCode, code)
			}

			if tc.expectedAnswer != "" {
				if len(rec.Msg.Answer) == 0 {
					t.Errorf("Expected an answer, but got none")
				} else {
					answer := rec.Msg.Answer[0]
					switch tc.queryType {
					case dns.TypeA:
						if a, ok := answer.(*dns.A); ok {
							if a.A.String() != tc.expectedAnswer {
								t.Errorf("Expected IP %s, but got %s", tc.expectedAnswer, a.A.String())
							}
						} else {
							t.Errorf("Expected A record, but got %T", answer)
						}
					case dns.TypeCNAME:
						if cname, ok := answer.(*dns.CNAME); ok {
							if dns.Fqdn(cname.Target) != dns.Fqdn(tc.expectedAnswer) {
								t.Errorf("Expected CNAME %s, but got %s", tc.expectedAnswer, cname.Target)
							}
						} else {
							t.Errorf("Expected CNAME record, but got %T", answer)
						}
					case dns.TypeTXT:
						if txt, ok := answer.(*dns.TXT); ok {
							if txt.Txt[0] != tc.expectedAnswer {
								t.Errorf("Expected TXT %s, but got %s", tc.expectedAnswer, txt.Txt[0])
							}
						} else {
							t.Errorf("Expected TXT record, but got %T", answer)
						}
					}
				}
			}
		})
	}
}
