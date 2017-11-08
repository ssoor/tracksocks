package socksd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ssoor/socks"
	"github.com/ssoor/fundadore/api"
	"github.com/ssoor/fundadore/log"
)

type UpstreamDialer struct {
	url string

	index   int
	lock    sync.Mutex
	dialers []socks.Dialer
}

func NewUpstreamDialer(url string) *UpstreamDialer {
	dialer := &UpstreamDialer{
		url:     url,
		dialers: []socks.Dialer{NewDecorateDirect(0, 0)}, // 原始连接,不经过任何处理
	}

	go dialer.backgroundUpdateServices()

	return dialer
}

func getSocksdSetting(url string) (setting Setting, err error) {
	var jsonData string

	if jsonData, err = api.GetURL(url); err != nil {
		return setting, errors.New(fmt.Sprint("Query setting interface failed, err: ", err))
	}

	if err = json.Unmarshal([]byte(jsonData), &setting); err != nil {
		return setting, errors.New("Unmarshal setting interface failed.")
	}

	return setting, nil
}

func buildUpstream(upstream Upstream, forward socks.Dialer) (socks.Dialer, error) {
	cipherDecorator := NewCipherConnDecorator(upstream.Crypto, upstream.Password)
	forward = NewDecorateClient(forward, cipherDecorator)

	switch strings.ToLower(upstream.Type) {
	case "socks5":
		{
			return socks.NewSocks5Client("tcp", upstream.Address, "", "", forward)
		}
	case "shadowsocks":
		{
			return socks.NewShadowSocksClient("tcp", upstream.Address, forward)
		}
	}
	return nil, errors.New("unknown upstream type" + upstream.Type)
}

func buildSetting(setting Setting) []socks.Dialer {
	var allForward []socks.Dialer
	for _, upstream := range setting.Upstreams {
		var forward socks.Dialer
		var err error
		forward = NewDecorateDirect(setting.DNSCacheTime, time.Duration(setting.DialTimeout))
		forward, err = buildUpstream(upstream, forward)
		if err != nil {
			log.Warning("failed to BuildUpstream, err:", err)
			continue
		}
		allForward = append(allForward, forward)
	}

	if len(allForward) == 0 {
		router := NewDecorateDirect(setting.DNSCacheTime, time.Duration(setting.DialTimeout))
		allForward = append(allForward, router)
	}

	return allForward
}

func (u *UpstreamDialer) backgroundUpdateServices() {
	var err error
	var setting Setting

	for {
		if setting, err = getSocksdSetting(u.url); nil != err {
			continue
		}

		log.Info("Setting messenger server information:")
		for _, upstream := range setting.Upstreams {
			log.Info("\tUpstream :", upstream.Address)
		}
		log.Info("\tDial timeout time :", setting.DialTimeout)
		log.Info("\tDNS cache timeout time :", setting.DNSCacheTime)
		log.Info("\tNext flush interval time :", setting.IntervalTime)

		u.lock.Lock()
		u.index = 0
		u.dialers = buildSetting(setting)
		u.lock.Unlock()

		time.Sleep(time.Duration(setting.IntervalTime) * time.Second)
	}
}

func (u *UpstreamDialer) CallNextDialer(network, address string) (conn net.Conn, err error) {
	u.lock.Lock()
	defer u.lock.Unlock()
	for {
		if 0 == len(u.dialers) {
			return socks.Direct.Dial(network, address)
		}

		if u.index++; u.index >= len(u.dialers) {
			u.index = 0
		}

		if conn, err = u.dialers[u.index].Dial(network, address); nil == err {
			break
		}

		switch err.(type) {
		case *net.OpError:
			if strings.EqualFold("dial", err.(*net.OpError).Op) {
				copy(u.dialers[u.index:], u.dialers[u.index+1:])
				u.dialers = u.dialers[:len(u.dialers)-1]

				log.Warning("Socks dial", network, address, "failed, delete current dialer:", err.(*net.OpError).Addr, ", err:", err)
				continue
			}
		}

		return nil, err
	}

	return conn, err
}

func (u *UpstreamDialer) Dial(network, address string) (conn net.Conn, err error) {
	if conn, err = u.CallNextDialer(network, address); nil != err {
		log.Warning("Upstream dial ", network, address, " failed, err:", err)
		return nil, err
	}

	return conn, nil
}
