package proxy

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ssoor/socks"
	"github.com/ssoor/fundadore/log"
)

type HTTPTransport struct {
	Rules *SRules
}

func (this *HTTPTransport) create502Response(req *http.Request, err error) (resp *http.Response) {

	resp = &http.Response{
		StatusCode: http.StatusBadGateway,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Request:    req,
		Header: http.Header{
			"X-Request-Error": []string{err.Error()},
		},
		ContentLength:    0,
		TransferEncoding: nil,
		Body:             ioutil.NopCloser(strings.NewReader("")),
		Close:            true,
	}

	return
}

func NewHTTPTransport(forward socks.Dialer, jsondata []byte) *HTTPTransport {
	transport := &HTTPTransport{
		Rules: NewSRules(forward),
	}

	if err := transport.Rules.ResolveJson(jsondata); nil != err {
		log.Error("Transport resolve json rule failed, err:", err)
	}

	return transport
}

func (this *HTTPTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {

	tranpoort, resp := this.Rules.ResolveRequest(req)

	if nil != resp {
		return resp, nil
	}

	req.Header.Del("X-Forwarded-For")
	req.Header.Set("Accept-Encoding", "gzip") // golang http response once support gzip

	if resp, err = tranpoort.RoundTrip(req); err != nil {
		if resp, err = tranpoort.RoundTrip(req); err != nil {
			log.Warning("tranpoort round trip:", req.URL.String(), ", err:", err)

			return this.create502Response(req, err), nil
		}
	}

	resp = this.Rules.ResolveResponse(req, resp)

	return
}
