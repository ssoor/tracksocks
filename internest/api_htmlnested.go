package internest

import (
	"net/http"

	"github.com/ssoor/webapi"
)

type HtmlNestedAPI struct {
	status int
	data   []byte
	header http.Header
}

func NewHtmlNestedAPI(nestedStatus int, nestedData []byte, nestedHeader map[string]string) *HtmlNestedAPI {
	htmlHeader := http.Header{}
	for name, value := range nestedHeader {
		htmlHeader.Add(name, value)
	}
	return &HtmlNestedAPI{
		status: nestedStatus,
		data:   nestedData,
		header: htmlHeader,
	}
}

func (nested HtmlNestedAPI) Get(values webapi.Values, request *http.Request) (int, interface{}, http.Header) {

	return nested.status, []byte(nested.data), nested.header // 默认IFrame
}
