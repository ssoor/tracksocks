package internest

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/ssoor/webapi"
	"github.com/ssoor/fundadore/log"
)

type LogAPI struct {
	url string
}

func NewLogAPI() *LogAPI {
	return &LogAPI{}
}

func (api LogAPI) Get(values webapi.Values, request *http.Request) (int, interface{}, http.Header) {
	var outstring string

	if logFile, err := os.OpenFile(log.GetFileName(), os.O_RDONLY, 0); nil == err {
		outstring = "<!DOCTYPE html><html><head><title>程序运行日志[" + log.GetFileName() + "]</title></head><style>html,body,textarea{height:99%;}</style><body><xmp>"

		if fd, err := ioutil.ReadAll(logFile); nil == err {
			outstring += string(fd)
		}

		outstring += "</xmp></body></html>"
	}

	return http.StatusOK, []byte(outstring), nil
}
