package trans_network

var (
	BroadcastServiceChannel        = "broadcast_local_services"
	ProxyHttpInitReq               = "/proxy_req/1.0"
	ProxyWsDataFromClientSide      = "/proxy_client_ws_data"
	ProxyWsDataFromServiceSide     = "/proxy_service_ws_data"
	ProxyHttpRespFromServiceSide   = "/proxy_service_http_data"
	ProxySocketInitReq             = "/proxy_client_socket_init"
	ProxySocketDataFromClientSide  = "/proxy_client_socket_data"
	ProxySocketDataFromServiceSide = "/proxy_service_socket_data"
)
