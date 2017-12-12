package redirect

import (
	"errors"
	"os"
	"time"

	"github.com/ssoor/socks"
	"github.com/ssoor/socks/upstream"
	"github.com/ssoor/fundadore/log"
	"github.com/ssoor/fundadore/api"
	"github.com/ssoor/fundadore/common"
	"github.com/ssoor/fundadore/config"
	"github.com/ssoor/fundadore/assistant"
	
	"github.com/ssoor/tracksocks/redirect/proxy"
)

const (
	PACListenPort uint16 = 44366
)

var (
	ErrorSettingQuery      error = errors.New("Query setting failed")
	ErrorSocksdCreate      error = errors.New("Create socksd failed")
	ErrorStartEncodeModule error = errors.New("Start encode module failed")
)

func runHTTPProxy(addr string, streamRouter socks.Dialer, transport *proxy.HTTPTransport, encode bool) {
	waitTime := float32(1)

	for {
		if encode {
			proxy.StartEncodeHTTPProxy(addr, streamRouter, transport)
		} else {
			proxy.StartHTTPProxy(addr, streamRouter, transport)
		}

		common.ChanSignalExit <- os.Kill

		waitTime += waitTime * 0.618
		log.Warning("Start http proxy unrecognized error, the terminal service will restart in", int(waitTime), "seconds ...")
		time.Sleep(time.Duration(waitTime) * time.Second)
	}
}

func runHTTPSProxy(addr string, streamRouter socks.Dialer, transport *proxy.HTTPTransport, encode bool) {
	waitTime := float32(1)

	for {
		if encode {
			proxy.StartEncodeHTTPSProxy(addr, streamRouter, transport)
		} else {
			proxy.StartHTTPSProxy(addr, streamRouter, transport)
		}

		common.ChanSignalExit <- os.Kill

		waitTime += waitTime * 0.618
		log.Warning("Start https proxy unrecognized error, the terminal service will restart in", int(waitTime), "seconds ...")
		time.Sleep(time.Duration(waitTime) * time.Second)
	}
}

func StartRedirect(account string, guid string, setting config.Redirect) (bool, error) {
	var err error = nil

	var connInternalIP string = "127.0.0.1"
	//connInternalIP, err := common.GetConnectIP("tcp", "www.baidu.com:80")

	log.Info("Set messenger encode mode:", setting.Encode)
	if err != nil {
		log.Error("Query connection ip failed:", err)
		return false, ErrorStartEncodeModule
	}

	srules, err := api.GetURL(setting.RulesURL)
	if err != nil {
		log.Errorf("Query srules interface failed, err: %s\n", err)
		return false, ErrorSettingQuery
	}

	router := upstream.NewUpstreamDialerByURL(setting.UpstreamsURL, 1 * 60 * 60)
	httpTransport := proxy.NewHTTPTransport(router, []byte(srules))

	addrHTTP, _ := common.SocketSelectAddr("tcp", connInternalIP)
	go runHTTPProxy(addrHTTP, router, httpTransport, setting.Encode)

	addrHTTPS, _ := common.SocketSelectAddr("tcp", connInternalIP)
	go runHTTPSProxy(addrHTTPS, router, httpTransport, setting.Encode)

	log.Info("Creating an internal server:")

	log.Info("\tHTTP Protocol:", addrHTTP)
	log.Info("\tHTTPS Protocol:", addrHTTPS)

	if err != nil {
		log.Error("Create messenger pac config failed, err:", err)
		return false, ErrorSocksdCreate
	}

	if setting.Encode {
		if err = proxy.AddCertificateToSystemStore(); nil != err {
			log.Warning("Add certificate to system stroe failed, err:", err)
		}
		
		log.Info("Setting redirect data share:")

		if host, port, err := common.SocketGetPortFormAddr(addrHTTP); nil != err {
			log.Warning("\tHTTP port parse failed, err:", err)
		}else{
			handle, err := assistant.SetBusinessData(1, 1, host, port)
			log.Info("\tHTTP handle:", handle, ", err:", err)
		}
		
		if host, port, err := common.SocketGetPortFormAddr(addrHTTPS); nil != err {
			log.Warning("\tHTTPS port parse failed, err:", err)
		}else{
			handle, err := assistant.SetBusinessData(2, 1, host, port)
			log.Info("\tHTTPS handle:", handle, ", err:", err)
		}
	}

	return true, nil
}
