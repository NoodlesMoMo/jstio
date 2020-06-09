package model

import (
	"errors"
	"git.sogou-inc.com/iweb/jstio/internel/logs"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/golang/protobuf/proto"
	"unsafe"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/mohae/deepcopy"
)

type ResourceBuilder struct {
	xdsCluster string

	fetcher EndpointFetcher
}

func MustNewResourceBuilder(xdsCluster string, fetcher EndpointFetcher) *ResourceBuilder {
	builder := &ResourceBuilder{
		xdsCluster: xdsCluster,
		fetcher:    fetcher,
	}
	return builder
}

func (rb *ResourceBuilder) BuildCustomAppEndpoints(app *Application) ([]cache.Resource, error) {

	var cfg []byte
	for _, res := range app.Resources {
		if res.ResType == ResourceTypeEndpoint {
			cfg = *(*[]byte)(unsafe.Pointer(&res.Config))
			break
		}
	}

	return rb.BuildCustomResource(app, ResourceTypeEndpoint, cfg)
}

// ClassifyResource: debug function
func (rb *ResourceBuilder) ClassifyResource(ress []cache.Resource) {
	for _, res := range ress {
		logs.Logger.Println("resource name:", proto.MessageName(res))
	}
}

func (rb *ResourceBuilder) BuildCustomResource(app *Application, resType ResourceType, cfg []byte) ([]cache.Resource, error) {
	tagLog := logs.FuncTaggedLoggerFactory()

	if resType == ResourceTypeEndpoint {
		ress, err := ValidationResource(resType, cfg)
		if err != nil {
			return nil, err
		}

		if err = app.loadApplicationXstream(); err != nil {
			return nil, err
		}

		selectors := make([]Selector, 0)
		for _, ups := range app.Upstream {
			selectors = append(selectors, ups)
		}

		endsDict, err := rb.fetcher.FetchEndpointsAgg(selectors...)
		if err != nil {
			return nil, err
		}

		for _, res := range ress {
			placeHoldLoadAssign, ok := res.(*v2.ClusterLoadAssignment)
			if !ok {
				return nil, errors.New("endpoint type but assert failed")
			}
			endpointClusterName := ProtocolHash(placeHoldLoadAssign.ClusterName)
			if endpointClusterName == app.Hash() {
				continue
			}

			placeHoldEnds := placeHoldLoadAssign.Endpoints
			placeHoldLbe := placeHoldEnds[0].LbEndpoints[0]

			ends, ok := endsDict[endpointClusterName]
			if !ok {
				tagLog("lookup etcd upstream").Errorln("miss! cluster name:", endpointClusterName)
				continue
			}

			lbes := make([]*envoy_api_v2_endpoint.LbEndpoint, len(ends))
			for idx, end := range ends {
				/* LbEndpoint has pointer, must deep copy */
				lbe := deepcopy.Copy(placeHoldLbe).(*envoy_api_v2_endpoint.LbEndpoint)
				sockAddress, ok := lbe.GetEndpoint().GetAddress().GetAddress().(*envoy_api_v2_core.Address_SocketAddress)
				if ok {
					sockAddress.SocketAddress.Address = end.Address
				} else {
					tagLog("place hold address").Errorln(endpointClusterName, "invalid socket address type")
				}
				lbes[idx] = lbe
			}
			placeHoldLoadAssign.Endpoints[0].LbEndpoints = lbes
		}

		return ress, err
	}

	return ValidationResource(resType, cfg)
}
