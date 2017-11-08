package socksd

import (
	"time"
	"crypto/tls"
	"net/http"

	"github.com/ssoor/socks"
	"github.com/ssoor/fundadore/log"
)

type HTTPHandler struct {
	scheme	  string
	proxy     *socks.HTTPProxy
}

func (h *HTTPHandler) ServeHTTP(rw http.ResponseWriter, request *http.Request) {
	request.URL.Scheme = h.scheme
	request.URL.Host = request.Host

	var handler http.Handler = h.proxy

	handler.ServeHTTP(rw, request)
}

func StartHTTPProxy(addr string, router socks.Dialer, tran *HTTPTransport) {
	handler := &HTTPHandler{
		scheme:    "HTTP",
		proxy:     socks.NewHTTPProxy("http", router, tran),
	}

	if err := http.ListenAndServe(addr, handler); nil != err {
		log.Error("Start HTTP proxy at ", addr, " failed, err:", err)
	}
}

func StartEncodeHTTPProxy(addr string, router socks.Dialer, tran *HTTPTransport) {
	if addr != "" {
		listener, err := NewEncodeListener(addr)
		if err != nil {
			log.Error("NewEncodeListener at ", addr, " failed, err:", err)
			return
		}

		defer listener.Close()

		handler := &HTTPHandler{
			scheme:    "HTTP",
			proxy:     socks.NewHTTPProxy("http", router, tran),
		}


		if err := http.Serve(listener, handler); nil != err {
			log.Error("Start HTTP encode proxy at ", addr, " failed, err:", err)
		}
		
	}
}


func HTTPSGetCertificate(clientHello *tls.ClientHelloInfo) (cert *tls.Certificate, err error) {
	if cert, err = QueryTlsCertificate(clientHello.ServerName); nil == err {
		return cert, err
	}

	return CreateTlsCertificate(nil, clientHello.ServerName, -(365 * 24 * time.Hour), 200)
}

func StartEncodeHTTPSProxy(addr string, router socks.Dialer, tran *HTTPTransport) {
	if addr != "" {
		listener, err := NewEncodeListener(addr)
		if err != nil {
			log.Error("NewEncodeListener at ", addr, " failed, err:", err)
			return
		}
		defer listener.Close()

		serverHTTPS := &http.Server{
			ErrorLog: log.Warn,
			TLSConfig: &tls.Config{
				GetCertificate: HTTPSGetCertificate,
			},
	
			Addr: addr,
			Handler: &HTTPHandler{
				scheme:    "HTTPS",
				proxy:     socks.NewHTTPProxy("https", router, tran),
			},
		}

	if err := serverHTTPS.ServeTLS(listener, "", ""); nil != err {
		log.Error("Start HTTPS encode proxy at ", addr, " failed, err:", err)
	}
	}
}

func StartHTTPSProxy(addr string, router socks.Dialer, tran *HTTPTransport) {
	serverHTTPS := &http.Server{
		ErrorLog: log.Warn,
		TLSConfig: &tls.Config{
			GetCertificate: HTTPSGetCertificate,
		},

		Addr: addr,
		Handler: &HTTPHandler{
			scheme:    "HTTPS",
			proxy:     socks.NewHTTPProxy("https", router, tran),
		},
	}

	if err := serverHTTPS.ListenAndServeTLS("", ""); nil != err {
		log.Error("Start HTTP proxy at ", addr, " failed, err:", err)
	}
}

