package model

import (
	"fmt"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"jstio/internel"
	"time"
)

type (
	EnvKind    = string
	StatusKind = string
	ReferKind  = string
)

type DataSourceMode = string

const (
	AdsMode  = `ads`
	XdsMode  = `xds`
	RestMode = `restful`
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
	localhost     = `127.0.0.1`
	listenAddress = `0.0.0.0`
	listenPort    = 80
)

const (
	EnvoyTypePrefix = `type.googleapis.com/envoy.api.v2.`
)

const (
	ProductEnvPrefix = `/conf/venus/planet/`
	DebugEnvPrefix   = `/conf/test/oneclass/`
)

const accessFormat = `[%START_TIME%] "%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% "%REQ(X-FORWARDED-FOR)%" "%REQ(USER-AGENT)%" "%REQ(X-REQUEST-ID)%" "%REQ(:AUTHORITY)%" "%UPSTREAM_HOST%"
`

var (
	refreshDelay = 500 * time.Millisecond
)

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

func WatchKeyPrefix() string {

	if internel.GetAfxMeta().DebugMode {
		return DebugEnvPrefix
	}

	return ProductEnvPrefix
}
