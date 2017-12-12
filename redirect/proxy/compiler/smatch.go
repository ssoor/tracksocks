package compiler

import (
	"errors"
	"regexp"
	"strings"

	"github.com/dlclark/regexp2"
)

type SMatch struct {
	template      []byte
	contextRegex  *regexp.Regexp
	contextRegex2 *regexp2.Regexp
}

var (
	ErrNotMatch           = errors.New("regular expression does not match")
	ErrUnrecognizedSMatch = errors.New("smatch resolution fails, unrecognized smatch expression")
)

func NewSMatch(rule string) (match SMatch, err error) {

	if rule[0] != 's' && rule[0] != 'S' {
		return match, ErrUnrecognizedSMatch // errors.New("invalid rule head: " + rule)
	}

	if rule[1] != '@' && rule[0] != '|' {
		return match, ErrUnrecognizedSMatch // errors.New("invalid character segmentation: " + rule)
	}

	split := strings.Split(rule, rule[1:2])

	if len(split) != 4 {
		return match, ErrUnrecognizedSMatch // errors.New("rule string incomplete or invalid: " + rule)
	}

	if match.contextRegex, err = regexp.Compile("(?" + split[3] + ")" + split[1]); nil != err {
		match.contextRegex2, err = regexp2.Compile("(?"+split[3]+")"+split[1], 0)
	}

	if err != nil {
		return match, ErrUnrecognizedSMatch // err
	}

	match.template = []byte(split[2])

	return match, nil
}

func (s *SMatch) Replace(src []byte) ([]byte, error) {
	if nil != s.contextRegex2 {
		if isMatch, err := s.contextRegex2.MatchString(string(src)); nil != err || false == isMatch { // 当出错时，返回 false
			return src, ErrNotMatch
		}

		dsc, err := s.contextRegex2.Replace(string(src), string(s.template), 0, 99999)
		return []byte(dsc), err
	}
	
	if isMatch := s.contextRegex.Match(src); false == isMatch { // 当出错时，返回 false
		return src, ErrNotMatch
	}

	return s.contextRegex.ReplaceAll(src, s.template), nil
}
