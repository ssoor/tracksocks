package socksd

import (
	"net"

	"errors"

	"github.com/ssoor/socks"
)

var ErrorUpStaeamDail error = errors.New("forward connect service failed.")

type DecorateClient struct {
	forward    socks.Dialer
	decorators []ConnDecorator
}

func NewDecorateClient(forward socks.Dialer, ds ...ConnDecorator) *DecorateClient {
	d := &DecorateClient{
		forward: forward,
	}
	d.decorators = append(d.decorators, ds...)
	return d
}

func (d *DecorateClient) Dial(network, address string) (net.Conn, error) {
	conn, err := d.forward.Dial(network, address)
	if err != nil {
		return nil, err
	}
	dconn, err := DecorateConn(conn, d.decorators...)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return dconn, nil
}
