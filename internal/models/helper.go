package models

import (
	"apron.network/gateway-p2p/internal"
	"bufio"
	"bytes"
	"github.com/valyala/fasthttp"
	"net/url"
	"path"
)

// RecoverClientRequest recover request serialized from client side to fasthttp request
func (svrReq *ApronServiceRequest) RecoverClientRequest() (*fasthttp.Request, error) {
	r := bufio.NewReader(bytes.NewReader(svrReq.RawRequest))
	httpReq := fasthttp.AcquireRequest()
	err := httpReq.Read(r)
	if err != nil {
		return nil, err
	}

	return httpReq, nil
}

func (svrReq *ApronServiceRequest) BuildHttpRequestToService(reqDetail *RequestDetail, httpReq *fasthttp.Request, serviceDetail *ApronService) *fasthttp.Request {
	// TODO: LB and multiple providers will be updated later
	baseUrlStr := serviceDetail.Providers[0].GetBaseUrl()
	serviceUrl, err := url.Parse(baseUrlStr)
	internal.CheckError(err)

	var clientSideQueryArgs fasthttp.Args
	httpReq.URI().QueryArgs().CopyTo(&clientSideQueryArgs)

	// Join path
	serviceUrl.Path = path.Join(serviceUrl.Path, string(reqDetail.ProxyRequestPath))
	httpReq.SetRequestURI(serviceUrl.String())

	clientSideQueryArgs.VisitAll(func(k, v []byte) {
		httpReq.URI().QueryArgs().AddBytesKV(k, v)
	})

	return httpReq
}
