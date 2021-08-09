package trans_network

import (
	"encoding/json"
	"log"
	"time"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"github.com/valyala/fasthttp"
)

// list all service including local and remote.
func (n *Node) listServiceHandler(ctx *fasthttp.RequestCtx) {
	log.Printf("[Service] List Available Service")
	rslt := make([]models.ApronService, 0, 100)

	n.mutex.Lock()
	log.Printf("[Service] List Service count: %+v\n", len(n.services))
	for _, v := range n.services {
		rslt = append(rslt, v)
	}
	n.mutex.Unlock()

	respBody, err := json.Marshal(rslt)
	internal.CheckError(err)
	ctx.Write(respBody)
}

// list all service peer including local and remote.
func (n *Node) listServicePeerHandler(ctx *fasthttp.RequestCtx) {
	log.Printf("[Service] List Available Service Peers")
	rslt := map[string]string{}

	n.mutex.Lock()
	log.Printf("[Service] List Service Peers count: %+v\n", len(n.servicePeerMapping))
	for k, v := range n.servicePeerMapping {
		rslt[k] = v.Pretty()
	}
	n.mutex.Unlock()

	respBody, err := json.Marshal(rslt)
	internal.CheckError(err)
	ctx.Write(respBody)
}

// Invoke RegisterLocalService to add service to local service list
// Publish service changes to all network via pubsub in BroadcastServiceChannel
func (n *Node) newOrUpdateServiceHandler(ctx *fasthttp.RequestCtx) {
	log.Printf("[Local Service] New OR Update Service")

	service := models.ApronService{}
	err := json.Unmarshal(ctx.Request.Body(), &service)
	internal.CheckError(err)

	n.RegisterLocalService(&service)
}

func (n *Node) deleteServiceHandler(ctx *fasthttp.RequestCtx) {
	log.Printf("[Service] Delete Service")

	service := models.ApronService{}
	err := json.Unmarshal(ctx.Request.Body(), &service)
	service.IsDeleted = true
	internal.CheckError(err)

	n.RegisterLocalService(&service)
}

func (n *Node) listLocalServiceHandler(ctx *fasthttp.RequestCtx) {
	log.Printf("[Local Service] List Available Service")
	rslt := make([]models.ApronService, 0, 100)

	n.mutex.Lock()
	for k, v := range n.servicePeerMapping {
		if v == n.selfID {
			rslt = append(rslt, n.services[k])
		}
	}
	n.mutex.Unlock()

	respBody, err := json.Marshal(rslt)
	internal.CheckError(err)
	ctx.Write(respBody)
}

func (n *Node) listRemoteServiceHandler(ctx *fasthttp.RequestCtx) {
	log.Printf("[Remote Service] List Available Service")
	rslt := make([]models.ApronService, 0, 100)

	n.mutex.Lock()
	for k, v := range n.servicePeerMapping {
		if v != n.selfID {
			rslt = append(rslt, n.services[k])
		}
	}
	n.mutex.Unlock()

	respBody, err := json.Marshal(rslt)
	internal.CheckError(err)
	ctx.Write(respBody)
}

func (n *Node) allUsageReportHandler(ctx *fasthttp.RequestCtx) {
	if rslt, err := n.serviceUsageRecordManager.ExportAllUsage(); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(err.Error())
	} else {
		usageRecordsJsonByte, err := json.Marshal(rslt)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(err.Error())
		}
		ctx.SetBody(usageRecordsJsonByte)
	}
}

func (n *Node) UpdatePeers() {
	peerRefreshTicker := time.NewTicker(time.Second)
	defer peerRefreshTicker.Stop()
	log.Printf("[Remote Service] Peers Monitor started")
	for {
		<-peerRefreshTicker.C
		availablePeers := n.ps.ListPeers(BroadcastServiceChannel)
		// log.Printf("availablePeers %++v", availablePeers)
		invaildService := make([]string, 0)
		n.mutex.Lock()
		for k, v := range n.servicePeerMapping {
			if v == n.selfID {
				continue
			}
			found := false
			for _, p := range availablePeers {
				if v == p {
					found = true
				}
			}

			if !found {
				invaildService = append(invaildService, k)
			}

		}

		// remove related services
		for _, service := range invaildService {
			log.Printf("[Remote Service] peer %s disconnected\n", n.servicePeerMapping[service])
			log.Printf("[Remote Service] remove service %s\n", n.services[service].Id)
			delete(n.services, service)
			delete(n.servicePeerMapping, service)
		}
		n.mutex.Unlock()

	}

}
