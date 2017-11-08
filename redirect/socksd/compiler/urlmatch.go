package compiler

import (
	"errors"
	"net/url"
	"strings"

	"regexp"

	"github.com/dlclark/regexp2"
)

type JSONURLMatch struct {
	Host  string   `json:"host"`
	Url   string   `json:"url"`
	Match []string `json:"match"`
}

type matchData struct {
	matchs    []SMatch
	urlRegex  *regexp.Regexp
	urlRegex2 *regexp2.Regexp
}
type URLMatch struct {
	data map[string][]matchData
}

func NewURLMatch() *URLMatch {
	return &URLMatch{data: make(map[string][]matchData)}
}

func (sc *URLMatch) AddMatchs(jsonMatchs JSONURLMatch) (err error) {
	var urlmatch matchData

	if urlmatch.urlRegex, err = regexp.Compile(jsonMatchs.Url); err != nil {
		if urlmatch.urlRegex2, err = regexp2.Compile(jsonMatchs.Url, 0); err != nil {
			return err
		}
	}

	for i := 0; i < len(jsonMatchs.Match); i++ {
		match, err := NewSMatch(jsonMatchs.Match[i])
		if err != nil {
			return err
		}

		urlmatch.matchs = append(urlmatch.matchs, match)
	}

	sc.data[jsonMatchs.Host] = append(sc.data[jsonMatchs.Host], urlmatch)
	return nil
}

func (sc *URLMatch) matchReplaces(md []matchData, url string, src []byte) (dst []byte, err error) {
	for _, urlmatch := range md {
		if nil != urlmatch.urlRegex2 {
			if isMatch, _ := urlmatch.urlRegex2.MatchString(url); false == isMatch { // 当出错时，返回 false
				continue
			}
		} else {
			if isMatch := urlmatch.urlRegex.MatchString(url); false == isMatch { // 当出错时，返回 false
				continue
			}
		}

		for _, match := range urlmatch.matchs {
			if dst, err = match.Replace(src); err == nil {
				return dst, nil
			}
		}
	}

	return src, errors.New("regular expression does not match")
}

func (sc *URLMatch) Replace(url *url.URL, src []byte) (dst []byte, err error) {
	host := strings.ToLower(url.Host)

	var exist bool
	var matchdatas []matchData

	matchdatas = sc.data[host] // 处理绝对匹配
	if dst, err = sc.matchReplaces(matchdatas, url.String(), src); nil == err {
		return
	}

	host = "." + host // 处理模糊匹配
	for i := 0; -1 != i; i = strings.IndexRune(host, '.') {
		host = host[i+1:]
		if matchdatas, exist = sc.data["."+host]; false == exist {
			continue
		}

		if dst, err = sc.matchReplaces(matchdatas, url.String(), src); nil == err {
			return
		}
	}

	matchdatas = sc.data["."] // 处理全局规则
	if dst, err = sc.matchReplaces(matchdatas, url.String(), src); nil == err {
		return
	}

	return src, errors.New("regular expression does not match")
}
