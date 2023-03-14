package trans_network

import (
	"encoding/json"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func (n *Node) StartMgmtApiServer() {
	// Init routers
	r := router.New()

	// Service related
	serviceRouter := r.Group("/service")
	serviceRouter.GET("/", n.listServiceHandler)
	serviceRouter.POST("/", n.newOrUpdateServiceHandler)
	serviceRouter.DELETE("/", n.deleteServiceHandler)
	serviceRouter.GET("/local", n.listLocalServiceHandler)
	serviceRouter.GET("/remote", n.listRemoteServiceHandler)
	serviceRouter.GET("/peers", n.listServicePeerHandler)

	n.logger.Info("management API server info",
		zap.String(EntityFieldName, EntityApronNode),
		zap.String("mgmt_addr", n.Config.MgmtAddr))
	fasthttp.ListenAndServe(n.Config.MgmtAddr, r.Handler)
}

// list all service including local and remote.
func (n *Node) listServiceHandler(ctx *fasthttp.RequestCtx) {

	rslt := make([]models.ApronService, 0, 100)

	n.mutex.Lock()
	n.logger.Debug("list available services",
		zap.String(EntityFieldName, EntityApronNode),
		zap.Int("service_count", len(n.services)),
		zap.Any("services", n.services),
	)
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
	rslt := map[string]string{}

	n.mutex.Lock()
	n.logger.Debug("list available service_peer mapping",
		zap.String(EntityFieldName, EntityApronNode),
		zap.Int("service_peer_mapping_count", len(n.servicePeerMapping)),
		zap.Any("service_peer_mapping", n.servicePeerMapping),
	)
	for k, v := range n.servicePeerMapping {
		rslt[k] = v.String()
	}
	n.mutex.Unlock()

	respBody, err := json.Marshal(rslt)
	internal.CheckError(err)
	ctx.Write(respBody)
}

// Invoke RegisterLocalService to add service to local service list
// Publish service changes to all network via pubsub in BroadcastServiceChannel
func (n *Node) newOrUpdateServiceHandler(ctx *fasthttp.RequestCtx) {
	service := models.ApronService{}
	err := json.Unmarshal(ctx.Request.Body(), &service)
	internal.CheckError(err)

	n.logger.Debug("new / update service",
		zap.String(EntityFieldName, EntityApronNode),
		zap.Any("service_detail", service))

	n.RegisterLocalService(&service)
}

func (n *Node) deleteServiceHandler(ctx *fasthttp.RequestCtx) {

	service := models.ApronService{}
	err := json.Unmarshal(ctx.Request.Body(), &service)
	service.IsDeleted = true
	internal.CheckError(err)

	n.logger.Debug("delete service",
		zap.String(EntityFieldName, EntityApronNode),
		zap.Any("service_detail", service))

	n.RegisterLocalService(&service)
}

func (n *Node) listLocalServiceHandler(ctx *fasthttp.RequestCtx) {
	rslt := make([]models.ApronService, 0, 100)

	n.mutex.Lock()
	for k, v := range n.servicePeerMapping {
		if v == n.selfID {
			rslt = append(rslt, n.services[k])
		}
	}
	n.mutex.Unlock()
	n.logger.Debug("list available local services",
		zap.String(EntityFieldName, EntityApronNode),
		zap.Any("service_local", rslt))

	respBody, err := json.Marshal(rslt)
	internal.CheckError(err)
	ctx.Write(respBody)
}

func (n *Node) listRemoteServiceHandler(ctx *fasthttp.RequestCtx) {
	rslt := make([]models.ApronService, 0, 100)

	n.mutex.Lock()
	for k, v := range n.servicePeerMapping {
		if v != n.selfID {
			rslt = append(rslt, n.services[k])
		}
	}
	n.mutex.Unlock()
	n.logger.Debug("list available remote services",
		zap.String(EntityFieldName, EntityApronNode),
		zap.Any("service_remote", rslt))

	respBody, err := json.Marshal(rslt)
	internal.CheckError(err)
	ctx.Write(respBody)
}
