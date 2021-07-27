package models

import (
	"apron.network/gateway-p2p/internal"
	"bufio"
	"bytes"
	"fmt"
	"github.com/valyala/fasthttp"
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

// DumpRequestToBytes dump fasthttp request to bytes.Buffer, then it will be sent to remote node
func DumpRequestToBytes(req *fasthttp.Request, buf *bytes.Buffer) error {
	w := bufio.NewWriter(buf)
	err := req.Write(w)
	w.Flush()
	return err
}

func ExtractRequestDetailFromFasthttpRequest(req *fasthttp.Request, srv *ApronService) (*RequestDetail, error) {
	detail := &RequestDetail{
		Host:        req.Host(),
		Path:        req.URI().Path(),
		ServiceName: internal.ServiceHostnameToIdByte(req.Host()),
		Method:      req.Header.Method(),
	}

	detail.Headers = make(map[string][]string)
	detail.QueryParams = make(map[string][]string)
	detail.FormParams = make(map[string][]string)

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
	fmt.Printf("Path with k: %+q, match rslt: %+v\n", pathWithKey, pathMatchResult)

	if len(pathMatchResult) == 1 && len(pathMatchResult[0]) == 4 {
		detail.Version, _ = strconv.ParseUint(string(pathMatchResult[0][1]), 10, 32)
		detail.UserKey = pathMatchResult[0][2]
		detail.ProxyRequestPath = pathMatchResult[0][3]
	}

	return detail, nil
}
