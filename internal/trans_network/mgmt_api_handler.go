package trans_network

import (
	"encoding/json"
	"log"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"github.com/valyala/fasthttp"
)

// list all service including local and remote.
func (n *Node) listServiceHandler(ctx *fasthttp.RequestCtx) {
	log.Printf("List Available Service")
	rslt := make([]models.ApronService, 0, 100)

	n.mutex.Lock()
	log.Printf("List Service count: %+v\n", len(n.services))
	for _, v := range n.services {
		rslt = append(rslt, v)
	}
	n.mutex.Unlock()

	respBody, err := json.Marshal(rslt)
	internal.CheckError(err)
	ctx.Write(respBody)
}

// Invoke RegisterLocalService to add service to local service list
// Publish service changes to all network via pubsub in BroadcastServiceChannel
func (n *Node) newOrUpdateServiceHandler(ctx *fasthttp.RequestCtx) {
	log.Printf("New OR Update Available Service")

	service := models.ApronService{}
	detail, err := models.ExtractRequestDetailFromFasthttpRequest(&ctx.Request, &service)
	internal.CheckError(err)
	err = json.Unmarshal(detail.RequestBody, &service)
	internal.CheckError(err)

	// check if new or update
	// currently only for debug
	// if _, ok := n.services[service.Id]; ok {
	// 	log.Printf("Update Available Service")

	// } else {
	// 	log.Printf("Create new Service")
	// }
	n.RegisterLocalService(&service)
}

func (n *Node) deleteServiceHandler(ctx *fasthttp.RequestCtx) {
	log.Printf("Delete Service")

	service := models.ApronService{}
	detail, err := models.ExtractRequestDetailFromFasthttpRequest(&ctx.Request, &service)
	internal.CheckError(err)
	err = json.Unmarshal(detail.RequestBody, &service)
	service.IsDeleted = true
	internal.CheckError(err)

	n.RegisterLocalService(&service)
}

func (n *Node) listLocalServiceHandler(ctx *fasthttp.RequestCtx) {
	log.Printf("List Local Available Service")
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
	log.Printf("List Remote Available Service")
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

}
