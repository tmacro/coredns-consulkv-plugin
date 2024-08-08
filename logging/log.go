package logging

import (
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var Log = clog.NewWithPlugin("consulkv")

func Set() {

}
