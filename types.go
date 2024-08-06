package consulkv

import (
	"encoding/json"

	"github.com/coredns/coredns/plugin"
	"github.com/hashicorp/consul/api"
)

type ConsulKV struct {
	Next    plugin.Handler
	Client  *api.Client
	Prefix  string
	Address string
	Token   string
	Zones   []string
}

type Record struct {
	TTL     *int `json:"ttl"`
	Records []struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	} `json:"records"`
}

type SRVRecord struct {
	Target   string `json:"target"`
	Port     uint16 `json:"port"`
	Priority uint16 `json:"priority"`
	Weight   uint16 `json:"weight"`
}

type SOARecord struct {
	MNAME   string `json:"mname"`
	RNAME   string `json:"rname"`
	SERIAL  uint32 `json:"serial"`
	REFRESH uint32 `json:"refresh"`
	RETRY   uint32 `json:"retry"`
	EXPIRE  uint32 `json:"expire"`
	MINIMUM uint32 `json:"minimum"`
}
