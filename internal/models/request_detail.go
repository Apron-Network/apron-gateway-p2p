package models

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/valyala/fasthttp"
	"log"
	"regexp"
	"strconv"
)

type RequestDetail struct {
	URI              *fasthttp.URI
	Host             []byte
	Path             []byte
	Method           []byte
	Headers          map[string][]string
	Cookies          map[string][]string
	QueryParams      map[string][]string
	FormParams       map[string][]string
	RequestBody      []byte
	Version          uint64
	ServiceName      []byte
	ServiceNameStr   string
	UserKey          []byte
	ApiKeyStr        string
	ProxyRequestPath []byte
}

var ProxyRequestPathPattern = regexp.MustCompile(`(?m)/v([1-9]\d{0,3})/([\w-]+)/?(.*)`)
var ServiceIdUserKeyPattern = regexp.MustCompile(`([\w-]{10})([\w-]+)`)

// DumpRequestToBytes dump fasthttp request to bytes.Buffer, then it will be sent to remote node
func DumpRequestToBytes(req *fasthttp.Request, buf *bytes.Buffer) error {
	w := bufio.NewWriter(buf)
	err := req.Write(w)
	w.Flush()
	return err
}

func ExtractServiceIdAndUserKey(combinedServiceIdUserKey []byte) ([]byte, []byte, error) {
	matchResult := ServiceIdUserKeyPattern.FindAllSubmatch(combinedServiceIdUserKey, -1)
	if len(matchResult) == 1 && len(matchResult[0]) == 3 {
		return matchResult[0][1], matchResult[0][2], nil
	} else {
		return nil, nil, errors.New("invalid path format")
	}
}

func ExtractRequestDetailFromFasthttpRequest(req *fasthttp.Request, detail *RequestDetail) error {
	var err error

	detail.Host = req.Host()
	detail.Path = req.URI().Path()
	detail.Method = req.Header.Method()
	detail.Headers = map[string][]string{}
	detail.QueryParams = map[string][]string{}
	detail.FormParams = map[string][]string{}

	req.Header.VisitAll(func(key, value []byte) {
		detail.Headers[string(key)] = append(detail.Headers[string(key)], string(value))
	})

	req.URI().QueryArgs().VisitAll(func(key, value []byte) {
		detail.QueryParams[string(key)] = append(detail.Headers[string(key)], string(value))
	})

	detail.RequestBody = req.Body()

	pathWithKey := detail.Path
	if bytes.HasPrefix(detail.Path, []byte("/ws")) {
		// Remove /ws prefix to match regexp for extracting version and key
		pathWithKey = pathWithKey[3:]
	}

	pathMatchResult := ProxyRequestPathPattern.FindAllSubmatch(pathWithKey, -1)
	log.Printf("Path with k: %+q, match rslt: %+q\n", pathWithKey, pathMatchResult)

	if len(pathMatchResult) == 1 && len(pathMatchResult[0]) == 4 {
		detail.Version, _ = strconv.ParseUint(string(pathMatchResult[0][1]), 10, 32)
		detail.ServiceName, detail.UserKey, err = ExtractServiceIdAndUserKey(detail.Path)
		if err != nil {
			return err
		}

		detail.ProxyRequestPath = pathMatchResult[0][3]
	} else {
		return errors.New("invalid path format")
	}

	return nil
}
