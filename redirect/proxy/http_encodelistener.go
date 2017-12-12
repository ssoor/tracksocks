package proxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/ssoor/fundadore/log"
)

const MaxHeaderSize = 4
const MaxBufferSize = 0x1000
const MaxEncodeSize = uint16(0xFFFF)

// NewHTTPLPProxy constructs one HTTPLPProxy
func NewEncodeListener(addr string) (*LPListener,error) {
	ln, err := net.Listen("tcp", addr)
	if  nil != err{
		return nil, err
	}

	return &LPListener{listener: ln}, nil
}

type ECipherConn struct {
	net.Conn
	rwc io.ReadWriteCloser

	isPass     bool
	beforeSend []byte

	decodeSize int
	decodeCode byte
	headBuffer [MaxHeaderSize]byte
}

func (this *ECipherConn) getEncodeSize(encodeHeader []byte) (int, error) {
	if encodeHeader[3] != (encodeHeader[0] ^ (encodeHeader[1] + encodeHeader[2])) {
		return 0, errors.New(fmt.Sprint("encryption header information check fails: ", encodeHeader[3], ",Unexpected value: ", (encodeHeader[0] ^ encodeHeader[1] + encodeHeader[2])))
	}

	return int(binary.BigEndian.Uint16(encodeHeader[1:3])), nil
}

func (econn *ECipherConn) Read(data []byte) (lenght int, err error) {

	if 0 != len(econn.beforeSend) { // 发送缓冲区中的数据
		bufSize := len(data)

		if bufSize > len(econn.beforeSend) {
			bufSize = len(econn.beforeSend)
		}

		err = nil
		lenght = copy(data, econn.beforeSend[:bufSize])

		econn.beforeSend = econn.beforeSend[bufSize:]

		//log.Info(string(data[:lenght]))
		//log.Info("Socksd header data is ", string(data[:lenght]))
		return
	}

	if econn.isPass { // 后续数据不用解密 ,直接调用原始函数 - isPass 由 readDecodeHeader() 函数设置
		lenght, err = econn.rwc.Read(data)

		if 0 != lenght && data[0] >= 'A' && data[0] <= 'z' {
			//log.Warning(string(data[:lenght]))
		}
		return
	}

	if 0 == econn.decodeSize { // 当前需要解密的数据为0，准备接受下一个加密头
		return econn.readDecodeHeader(data)
	}

	///////////////////////////////////////////////////////////////////////////////////////////
	//

	econn.beforeSend = make([]byte, econn.decodeSize)
	if lenght, err = io.ReadFull(econn.rwc, econn.beforeSend); nil != err { // 检测数据包是否为加密包或者有效的 HTTP 包
		if io.ErrUnexpectedEOF == err {
			err = io.EOF
		}

		if io.EOF != err {
			log.Warning("Socket full reading failed, current read data size:", lenght, ", need read size:", econn.decodeSize, " err is:", err)
		}

	}

	for i := 0; i < len(econn.beforeSend); i++ { // econn.decodeSize
		econn.beforeSend[i] ^= econn.decodeCode | 0x80
	}

	econn.decodeSize = 0

	return 0, nil
}

// 加密类型
const (
	HeaderGet     = 0xCD
	HeaderPost    = 0xDC
	HeaderConnect = 0x00

	HeaderPut    = 0xF0
	HeaderHead   = 0xF1
	HanderTrace  = 0xF2
	HanderDelect = 0xF3

	HanderBinary = 0xFF
)

func (econn *ECipherConn) isDecodeHeader(data byte) bool {
	switch data {
	case HeaderGet: // GET
	case HeaderPost: // POST
	case HeaderConnect: // CONNNECT
	case HeaderPut: // PUT
	case HeaderHead: // HEAD
	case HanderTrace: // TRACE
	case HanderDelect: // DELECT
	case HanderBinary:
	default:
		return false
	}

	return true
}

func (this *ECipherConn) readDecodeHeader(data []byte) (lenght int, err error) {
	this.headBuffer[0] = 0
	this.headBuffer[1] = 0
	this.headBuffer[2] = 0
	this.headBuffer[3] = 0

	if lenght, err = this.rwc.Read(this.headBuffer[:1]); 1 != lenght {
		return
	}

	this.isPass = true // 一个新的数据包,默认不需要解密，直接放过
	if false == this.isDecodeHeader(this.headBuffer[0]) {
		this.beforeSend = this.headBuffer[:1] // 数据需要发送

		if this.headBuffer[0] >= 'A' && this.headBuffer[0] <= 'z' {
			log.Warning("Socket decode check failed, current encode type is", this.headBuffer[0])
		}
		return 0, nil
	}

	///////////////////////////////////////////////////////////////////////////////////////////
	//

	if lenght, err = io.ReadFull(this.rwc, this.headBuffer[1:]); nil != err { // 检测数据包是否为加密包或者有效的 HTTP 包
		if io.ErrUnexpectedEOF == err {
			err = io.EOF
		}

		if io.EOF == err {
			this.isPass = false
		} else {
			log.Warning("Socket full reading failed, current read data:", string(this.headBuffer[:1+lenght]), "(", 1+lenght, "), need read size:", MaxHeaderSize, " err is:", err)
		}
		return copy(data, this.headBuffer[:lenght]), err
	}

	this.beforeSend = this.headBuffer[:MaxHeaderSize] // make([]byte, MaxHeaderSize) // 数据需要发送

	if lenght, err = this.getEncodeSize(this.headBuffer[:MaxHeaderSize]); nil != err || lenght > int(MaxEncodeSize) {
		return copy(data, this.headBuffer[:MaxHeaderSize]), nil
	}

	this.isPass = false // 数据需要解密
	this.decodeSize = lenght
	this.decodeCode = this.headBuffer[3]

	switch this.headBuffer[0] {
	case HeaderGet: // GET
		this.beforeSend[0] = 'G'
		this.beforeSend[1] = 'E'
		this.beforeSend[2] = 'T'
		this.beforeSend[3] = ' '
	case HeaderPost: // POST
		this.beforeSend[0] = 'P'
		this.beforeSend[1] = 'O'
		this.beforeSend[2] = 'S'
		this.beforeSend[3] = 'T'
	case HeaderConnect: // CONNNECT
		this.beforeSend[0] = 'C'
		this.beforeSend[1] = 'O'
		this.beforeSend[2] = 'N'
		this.beforeSend[3] = 'N'
	case HeaderPut: // PUT
		this.beforeSend[0] = 'P'
		this.beforeSend[1] = 'U'
		this.beforeSend[2] = 'T'
		this.beforeSend[3] = ' '
	case HeaderHead: // HEAD
		this.beforeSend[0] = 'H'
		this.beforeSend[1] = 'E'
		this.beforeSend[2] = 'A'
		this.beforeSend[3] = 'D'
	case HanderTrace: // TRACE
		this.beforeSend[0] = 'T'
		this.beforeSend[1] = 'R'
		this.beforeSend[2] = 'A'
		this.beforeSend[3] = 'C'
	case HanderDelect: // DELECT
		this.beforeSend[0] = 'D'
		this.beforeSend[1] = 'E'
		this.beforeSend[2] = 'L'
		this.beforeSend[3] = 'E'
	case HanderBinary:
		this.beforeSend = nil
	default:
		log.Warning("Unknown socksd encode type:", this.headBuffer[0], ", encode len: ", this.decodeSize, "\n")
	}

	if nil != this.beforeSend {
		log.Warning("Old socksd encode type:", this.headBuffer[0], ", encode len: ", this.decodeSize, "\n")
	}

	//log.Infof("Socksd encode code: % 5d , encode len: %d\n", this.decodeCode, this.decodeSize)

	return 0, nil
}

func (c *ECipherConn) Write(data []byte) (int, error) {
	return c.rwc.Write(data)
}

func (c *ECipherConn) Close() error {
	err := c.Conn.Close()
	c.rwc.Close()
	return err
}

type LPListener struct {
	listener net.Listener
}

func (this *LPListener) Accept() (c net.Conn, err error) {
	conn, err := this.listener.Accept()

	if err != nil {
		return nil, err
	}

	return &ECipherConn{
		Conn: conn,
		rwc:  conn,

		isPass:     false,
		beforeSend: nil,
	}, nil
}

func (this *LPListener) Close() error {
	return this.listener.Close()
}

func (this *LPListener) Addr() net.Addr {
	return this.listener.Addr()
}
