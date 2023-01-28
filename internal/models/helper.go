package models

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/url"
	"path"
	"strings"

	"github.com/valyala/fasthttp"
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

func (svrReq *ApronServiceRequest) BuildHttpRequestToService(reqDetail *RequestDetail, httpReq *fasthttp.Request, serviceDetail *ApronService) (*fasthttp.Request, error) {
	// TODO: LB and multiple providers will be updated later
	provider := serviceDetail.Providers[0]
	serviceUrl, err := url.Parse(fmt.Sprintf("%s://%s", provider.GetSchema(), provider.GetBaseUrl()))
	if err != nil {
		return nil, err
	}

	var clientSideQueryArgs fasthttp.Args
	httpReq.URI().QueryArgs().CopyTo(&clientSideQueryArgs)

	// Join path
	serviceUrl.Path = path.Join(serviceUrl.Path, string(reqDetail.ProxyRequestPath))
	httpReq.SetRequestURI(serviceUrl.String())

	clientSideQueryArgs.VisitAll(func(k, v []byte) {
		httpReq.URI().QueryArgs().AddBytesKV(k, v)
	})

	return httpReq, nil
}

func ExtractServiceInfoFromRequestID(requestId string) (string, string) {
	rslt := strings.SplitN(requestId, ".", 3)
	if len(rslt) != 3 {
		log.Panicf(fmt.Errorf("invalid format of request id (%s), please contract admin", requestId).Error())
	}
	return rslt[0], rslt[1]
}
