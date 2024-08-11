package consulkv

import (
	"net"

	"github.com/miekg/dns"
)

type ResponseWriterWrapper struct {
	WrappedMsg *dns.Msg
}

func (w *ResponseWriterWrapper) WriteMsg(res *dns.Msg) error {
	w.WrappedMsg.Answer = append(w.WrappedMsg.Answer, res.Answer...)
	return nil
}

func (w *ResponseWriterWrapper) LocalAddr() net.Addr {
	return nil
}

func (w *ResponseWriterWrapper) RemoteAddr() net.Addr {
	return nil
}

func (w *ResponseWriterWrapper) Write([]byte) (int, error) {
	return 0, nil
}

func (w *ResponseWriterWrapper) Close() error {
	return nil
}

func (w *ResponseWriterWrapper) TsigStatus() error {
	return nil
}

func (w *ResponseWriterWrapper) TsigTimersOnly(bool) {

}

func (w *ResponseWriterWrapper) Hijack() {

}
