package trans_network

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"encoding/json"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"log"
)

func (n *Node) StartMgmtApiServer() {
	// Init routers
	router := router.New()

	// Service related
	serviceRouter := router.Group("/service")
	serviceRouter.GET("/", n.listServiceHandler)
	serviceRouter.POST("/", n.newOrUpdateServiceHandler)
	serviceRouter.DELETE("/", n.deleteServiceHandler)
	serviceRouter.GET("/local", n.listLocalServiceHandler)
	serviceRouter.GET("/remote", n.listRemoteServiceHandler)
	serviceRouter.GET("/peers", n.listServicePeerHandler)

	log.Printf("Management API Server: %s\n", n.Config.MgmtAddr)
	fasthttp.ListenAndServe(n.Config.MgmtAddr, router.Handler)
}

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
