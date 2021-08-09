package models

import (
	"apron.network/gateway-p2p/internal"
	"bufio"
	"bytes"
	"github.com/valyala/fasthttp"
	"net/url"
	"path"
)

func (svrReq *ApronServiceRequest) BuildHttpRequest(serviceDetail *ApronService) *fasthttp.Request {
	r := bufio.NewReader(bytes.NewReader(svrReq.RawRequest))
	httpReq := fasthttp.AcquireRequest()
	err := httpReq.Read(r)
	internal.CheckError(err)

	reqDetail, err := ExtractRequestDetailFromFasthttpRequest(httpReq)
	// log.Printf("Service detail: %+v", reqDetail)

	// TODO: LB and multiple providers will be updated later
	baseUrlStr := serviceDetail.Providers[0].GetBaseUrl()
	serviceUrl, err := url.Parse(baseUrlStr)
	internal.CheckError(err)

	// Join path
	serviceUrl.Path = path.Join(serviceUrl.Path, string(reqDetail.ProxyRequestPath))

	httpReq.SetRequestURI(serviceUrl.String())

	return httpReq
}
