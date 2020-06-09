package model

import (
	"fmt"
	"git.sogou-inc.com/iweb/jstio/internel"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
)

type (
	EnvKind    = string
	StatusKind = string
	ReferKind  = string
)

type DataSourceMode = string

const (
	SOUGOSLD = `odin.sogou`
)

const (
	AdsMode  = `ads`
	XdsMode  = `xds`
	RestMode = `restful`
)

const (
	ProtocolHTTP = `http`
	ProtocolGRPC = `grpc`
	ProtocolUsr1 = `usr1`
	ProtocolUsr2 = `usr2`
)

const (
	EnvDevelopment = `development`
	EnvTest        = `test`
	EnvGray        = `gray`
	EnvProduct     = `product`
)

const (
	StatusPending = `pending`
	StatusEnable  = `enable`
	StatusDisable = `disable`
	StatusDeleted = `deleted`
)

const (
	ReferKindNone       = `none`
	ReferKindUpstream   = `upstream`
	ReferKindDownstream = `downstream`
)

const (
	Localhost     = `127.0.0.1`
	ListenAddress = `0.0.0.0`
	ListenPort    = 80
)

const (
	EnvoyTypePrefix = `type.googleapis.com/envoy.api.v2.`
)

const AccessFormat = `[%START_TIME%] "%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% "%REQ(X-FORWARDED-FOR)%" "%REQ(USER-AGENT)%" "%REQ(X-REQUEST-ID)%" "%REQ(:AUTHORITY)%" "%UPSTREAM_HOST%"
`

var (
	Zero = struct{}{}
)

type PodInfo struct {
	HostName string `json:"host_name"`
	IP       string `json:"ip"`
}

type Endpoint struct {
	Address string
	Port    uint32
}

func (e *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", e.Address, e.Port)
}

type Selector interface {
	Hash() string
	SelectorFormat() string
	SelectorScan(key string) error
}

type EndpointFetcher interface {
	FetchEndpoints(selector Selector) ([]Endpoint, error)
	FetchEndpointsAgg(selectors ...Selector) (map[string][]Endpoint, error)
}

type XdsResource struct {
	Endpoints []cache.Resource
	Clusters  []cache.Resource
	Routers   []cache.Resource
	Listener  []cache.Resource
}

func WatchKeyPrefix() []string {
	return internel.GetAfxOption().GetETCDPrefixKeys()
}
