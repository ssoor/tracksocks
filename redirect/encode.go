package redirect

import (
	"encoding/binary"
	"net"

	"github.com/ssoor/fundadore/common"
	"github.com/ssoor/fundadore/assistant"
)

func SocketCreateSockAddr(addr string) (addrSocket assistant.SOCKADDR_IN, err error) {
	var port uint16
	var host string
	addrSocket.Sin_family = 2 // AF_INET

	if host, port, err = common.SocketGetPortFormAddr(addr); nil != err {
		return addrSocket, err
	}

	binary.BigEndian.PutUint16(addrSocket.Sin_port[0:], port)

	ipv4 := net.ParseIP(host)

	ipv4 = ipv4.To4()
	buff := make([]byte, 0)

	buff = append(buff, ipv4...)

	copy(addrSocket.Sin_addr[0:], buff)

	return addrSocket, nil
}
