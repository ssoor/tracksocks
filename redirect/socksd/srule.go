package socksd

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	//	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/ssoor/socks"
	"github.com/ssoor/fundadore/log"
	
	"github.com/ssoor/tracksocks/redirect/socksd/compiler"
)

const (
	Rewrite_URL = iota
	Redirect_URL
	Rewrite_HTML
	Rewrite_JaveScript
	FastRedirect_URL
)

type internalJSONURLMatch struct {
	compiler.JSONURLMatch
	Type int `json:"type"`
}

type JSONSRule struct {
	Compiler []internalJSONURLMatch `json:"compilers"`
}

type JSONLimits struct {
	MaxResponseContentLen int64 `json:"max_response_content_len"`
}

type JSONRules struct {
	Local  bool        `json:"local"`
	Limits JSONLimits  `json:"limits"`
	SRules []JSONSRule `json:"srules"`
}

type SRules struct {
	local    bool
	limits   JSONLimits
	urlMatch map[int]*compiler.URLMatch

	tranpoort_local  *http.Transport
	tranpoort_remote *http.Transport
}

func NewSRules(forward socks.Dialer) *SRules {

	return &SRules{
		tranpoort_remote: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return forward.Dial(network, addr)
			},
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		tranpoort_local: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return socks.Direct.Dial(network, addr)
			},
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		urlMatch: make(map[int]*compiler.URLMatch),
	}
}

func (s *SRules) ResolveJson(data []byte) (err error) {

	jsonRules := JSONRules{}

	if err = json.Unmarshal(data, &jsonRules); err != nil {
		return err
	}

	s.local = jsonRules.Local
	s.limits = jsonRules.Limits

	if false == s.local {
		s.tranpoort_local = s.tranpoort_remote
	}

	for i := 0; i < len(jsonRules.SRules); i++ {
		for j := 0; j < len(jsonRules.SRules[i].Compiler); j++ {
			if err := s.Add(jsonRules.SRules[i].Compiler[j]); nil != err {
				return err
			}
		}
	}

	return nil
}

func (s *SRules) Add(internalMatch internalJSONURLMatch) (err error) {
	var match compiler.JSONURLMatch

	match.Url = internalMatch.Url
	match.Host = internalMatch.Host
	match.Match = internalMatch.Match

	if nil == s.urlMatch[internalMatch.Type] {
		s.urlMatch[internalMatch.Type] = compiler.NewURLMatch()
	}

	err = s.urlMatch[internalMatch.Type].AddMatchs(match)

	for i := 0; i < len(match.Match); i++ {
		log.Info("Sign up routing:", err, internalMatch.Type, fmt.Sprintf("%s(%s)", match.Host, match.Url), match.Match[i])
	}

	err = nil // 规则配置错误不影响其他规则运行

	return err
}

func (s *SRules) Replace(matchType int, url *url.URL, src []byte) (dst []byte, err error) {
	if nil == s.urlMatch[matchType] {
		return src, errors.New("Rule not found.")
	}

	return s.urlMatch[matchType].Replace(url, src)
}

func (s *SRules) replaceURL(matchType int, srcurl *url.URL) (dsturl *url.URL, err error) {
	var dststr []byte
	if dststr, err = s.Replace(matchType, srcurl, []byte(srcurl.String())); err != nil {
		return nil, err
	}

	if dsturl, err = url.Parse(string(dststr)); err != nil {
		return nil, err
	}

	return dsturl, nil
}

func (s *SRules) GetRewriteURL(req *http.Request) (dst *url.URL, err error) {
	var dststr []byte
	if dststr, err = s.Replace(Rewrite_URL, req.URL, []byte(req.URL.String())); err != nil {
		return nil, err
	}

	return url.Parse(string(dststr))
}

func (s *SRules) GetRedirectURL(req *http.Request) (dst *url.URL, err error) {
	var dststr []byte
	if dststr, err = s.Replace(Redirect_URL, req.URL, []byte(req.URL.String())); err != nil {
		return nil, err
	}

	return url.Parse(string(dststr))
}

func (s *SRules) GetFastRedirectURL(req *http.Request) (dst *url.URL, err error) {
	var dststr []byte
	if dststr, err = s.Replace(FastRedirect_URL, req.URL, []byte(req.URL.String())); err != nil {
		return nil, err
	}

	if unescape, err := url.QueryUnescape(string(dststr)); err == nil {
		dststr = []byte(unescape)
	}

	return url.Parse(string(dststr))
}

func (s *SRules) ResolveRequest(req *http.Request) (tran *http.Transport, resp *http.Response) {
	var err error
	var dsturl *url.URL
	tran = s.tranpoort_local

	if dsturl, err = s.GetFastRedirectURL(req); nil == err {
		if false == strings.EqualFold(req.URL.String(), dsturl.String()) {
			log.Info("Redirect request", req.URL, "to", dsturl)

			req.URL = dsturl

			tran = nil
			resp = s.createRedirectResponse(dsturl.String(), req)
		} else {
			log.Info("Sending request", req.URL, "is remote.")

			resp = nil
			tran = s.tranpoort_remote
		}
	} else if dsturl, err = s.GetRedirectURL(req); nil == err {
		if false == strings.EqualFold(req.URL.String(), dsturl.String()) {
			log.Info("Redirect request", req.URL, "to", dsturl)

			req.URL = dsturl

			tran = nil
			resp = s.createRedirectResponse(dsturl.String(), req)
		} else {
			log.Info("Sending request", req.URL, "is remote.")

			resp = nil
			tran = s.tranpoort_remote
		}
	} else if dsturl, err = s.GetRewriteURL(req); nil == err {
		if strings.EqualFold(req.URL.Host, dsturl.Host) {
			log.Info("Rewrite request", req.URL, "to", dsturl)

			req.URL = dsturl

			resp = nil
			tran = s.tranpoort_remote
		} else {
			log.Error("Rewrite request", req.URL, "to", dsturl, "failed: Unauthorized jump, the host does not match")
		}
	}

	return tran, resp
}

func (s *SRules) GetResponseBody(resp *http.Response) (html []byte, err error) {
	defer func() {
		if nil != err {
			log.Warning("Rewrite javescript failed, err:", err)
		}
	}()

	bodyReader := bufio.NewReader(resp.Body)
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		var read *gzip.Reader
		if read, err = gzip.NewReader(resp.Body); nil != err {
			err = errors.New(fmt.Sprint("create gzip reader error:", err))
			return
		}

		bodyReader = bufio.NewReader(read)
		resp.Header.Del("Content-Encoding")
	}

	//dumpdata, _ := httputil.DumpResponse(resp, true)
	//log.Println(string(dumpdata))

	var bodyBuf bytes.Buffer
	_, err = bodyBuf.ReadFrom(bodyReader)

	if io.ErrUnexpectedEOF == err {
		err = nil
	}

	return bodyBuf.Bytes(), nil
}

func (s *SRules) GetRewriteHTML(req *http.Request, resp *http.Response) (dst []byte, err error) {
	if resp_type := resp.Header.Get("Content-Type"); false == strings.Contains(strings.ToLower(resp_type), "text/html") {
		return []byte{}, errors.New("Content type mismatch.")
	}

	var newHTML []byte
	if newHTML, err = s.GetResponseBody(resp); nil != err {
		// resp.Body 缓冲区被转换成 bytes.Buffer 之后，必须通过新内容形式返回，因为原始的 resp.Body 已经被破坏
		return newHTML, nil
	}

	if data, err := s.Replace(Rewrite_HTML, resp.Request.URL, newHTML); nil == err {
		newHTML = data
		log.Info("Injection html", resp.Request.URL.String(), " successed, old size", resp.ContentLength, ", new size", len(newHTML))
	}

	return newHTML, nil // 不能返回错误
}

func (s *SRules) GetRewriteJaveScript(req *http.Request, resp *http.Response) (dst []byte, err error) {
	if resp_type := resp.Header.Get("Content-Type"); false == strings.Contains(strings.ToLower(resp_type), "text/javascript") && false == strings.Contains(strings.ToLower(resp_type), "application/javascript") && false == strings.Contains(strings.ToLower(resp_type), "application/x-javascript") {
		return []byte{}, errors.New("Content type mismatch.")
	}

	var newHTML []byte
	if newHTML, err = s.GetResponseBody(resp); nil != err {
		// resp.Body 缓冲区被转换成 bytes.Buffer 之后，必须通过新内容形式返回，因为原始的 resp.Body 已经被破坏
		return newHTML, nil
	}

	if data, err := s.Replace(Rewrite_JaveScript, resp.Request.URL, newHTML); nil == err {
		newHTML = data
		log.Info("Injection html", resp.Request.URL.String(), " successed, old size is", resp.ContentLength)
	}

	return newHTML, nil // 不能返回错误
}

func (s *SRules) ResolveResponse(req *http.Request, resp *http.Response) *http.Response {
	var err error
	var html []byte

	if resp.ContentLength == 0 || resp.ContentLength > s.limits.MaxResponseContentLen {
		return resp
	}

	if html, err = s.GetRewriteHTML(req, resp); nil == err {
		log.Info("Resolve response url(html):", resp.Request.URL)
	} else if html, err = s.GetRewriteJaveScript(req, resp); nil == err {
		log.Info("Resolve response url(javascript):", resp.Request.URL)
	} else {
		return resp
	}

	if -1 != resp.ContentLength {
		resp.ContentLength = int64(len([]byte(html)))
		resp.Header.Set("Content-Length", strconv.Itoa(len([]byte(html))))
	}

	oldBody := resp.Body
	defer oldBody.Close()

	resp.Body = ioutil.NopCloser(bytes.NewReader(html))

	return resp
}

func (this *SRules) createRedirectResponse(url string, req *http.Request) (resp *http.Response) {

	resp = &http.Response{
		StatusCode: http.StatusFound,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Request:    req,
		Header: http.Header{
			"Location": []string{url},
		},
		ContentLength:    0,
		TransferEncoding: nil,
		Body:             ioutil.NopCloser(strings.NewReader("")),
		Close:            true,
	}

	return
}
