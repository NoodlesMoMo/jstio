package model

import (
	"errors"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"jstio/internel"
	"jstio/internel/logs"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	xcluster "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	alg "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	httpConnMgr "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/gogo/protobuf/types"
	"github.com/mohae/deepcopy"
)

var (
	/*
		https://www.envoyproxy.io/docs/envoy/latest/faq/disable_circuit_breaking.html?highlight=max_connections
	*/
	unLimitThresholds = &types.UInt32Value{
		Value: 1000000000,
	}

	// retry default setting values
	defaultRetriesNum = &types.UInt32Value{
		Value: 3,
	}

	defaultRetryTimeout = 2 * time.Second
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

func (rb *ResourceBuilder) BuildFastAppEndpoints(app *Application) ([]cache.Resource, error) {
	var (
		err  error
		ress []cache.Resource
	)

	tagLog := logs.FuncTaggedLoggerFactory()

	selfEnds := make([]endpoint.LbEndpoint, 0)
	for _, port := range app.SelfPorts() {
		selfEnds = append(selfEnds, endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol:      core.TCP,
								Address:       localhost,
								PortSpecifier: &core.SocketAddress_PortValue{PortValue: port},
							},
						},
					},
				},
			},
		})
	}

	ress = append(ress, &v2.ClusterLoadAssignment{
		ClusterName: app.Domain(),
		Endpoints:   []endpoint.LocalityLbEndpoints{{LbEndpoints: selfEnds}},
	})

	for _, referApp := range app.Upstream {
		lbEnds := make([]endpoint.LbEndpoint, 0)
		ends, e := rb.fetcher.FetchEndpoints(referApp)
		if e != nil {
			err = e
			tagLog(`fetch endpoint`).Errorln("app:", referApp.Domain(), "err:", e)
		} else {
			tagLog(`fetch endpoint`).Warning(referApp.Domain(), "ends:", ends)
		}

		for _, end := range ends {
			item := endpoint.LbEndpoint{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol:      core.TCP,
									Address:       end.Address,
									PortSpecifier: &core.SocketAddress_PortValue{PortValue: end.Port},
								},
							},
						},
					},
				},
			}

			lbEnds = append(lbEnds, item)
		}

		ress = append(ress, &v2.ClusterLoadAssignment{
			ClusterName: referApp.Domain(),
			Endpoints:   []endpoint.LocalityLbEndpoints{{LbEndpoints: lbEnds}},
		})
	}

	return ress, err
}

func (rb *ResourceBuilder) BuildFastAppCluster(app *Application) ([]cache.Resource, error) {

	edsSource := rb.makeDataSource(internel.GetAfxMeta().PushMode) // FIXME: mode as param ???

	clusterNames := []string{app.Domain()}
	for _, up := range app.Upstream {
		clusterNames = append(clusterNames, up.Domain())
	}

	ress := make([]cache.Resource, 0)
	for _, name := range clusterNames {
		cluster := &v2.Cluster{
			Name:                 name,
			ConnectTimeout:       3 * time.Second,
			ClusterDiscoveryType: &v2.Cluster_Type{Type: v2.Cluster_EDS},
			EdsClusterConfig: &v2.Cluster_EdsClusterConfig{
				EdsConfig: edsSource,
			},
			CircuitBreakers: &xcluster.CircuitBreakers{
				Thresholds: []*xcluster.CircuitBreakers_Thresholds{{
					Priority:           core.RoutingPriority_HIGH,
					MaxConnections:     unLimitThresholds,
					MaxPendingRequests: unLimitThresholds,
					MaxRequests:        unLimitThresholds,
					MaxRetries:         &types.UInt32Value{Value: 3},
				}},
			},
		}
		ress = append(ress, cluster)
	}

	return ress, nil
}

func (rb *ResourceBuilder) BuildFastAppRoutes(app *Application) ([]cache.Resource, error) {

	// build self, redirect action
	var vHosts = []route.VirtualHost{
		{
			Name:    app.Domain(),
			Domains: []string{app.Domain()},
			Routes: []route.Route{{
				Match: route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: app.Domain(),
						},
					},
				},
			}},
			IncludeRequestAttemptCount: true,
			RetryPolicy: &route.RetryPolicy{
				RetryOn:       `reset,connect-failure,refused-stream`,
				NumRetries:    defaultRetriesNum,
				PerTryTimeout: &defaultRetryTimeout,
				RetryHostPredicate: []*route.RetryPolicy_RetryHostPredicate{{
					Name: `envoy.retry_host_predicates.previous_hosts`,
				}},
				RetryPriority: &route.RetryPolicy_RetryPriority{
					Name: `envoy.retry_priorities.previous_priorities`,
					ConfigType: &route.RetryPolicy_RetryPriority_Config{
						Config: &types.Struct{
							Fields: map[string]*types.Value{
								"update_frequency": {
									Kind: &types.Value_NumberValue{NumberValue: 2},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, a := range app.Upstream {
		vHosts = append(vHosts, route.VirtualHost{
			Name:    a.Domain(),
			Domains: []string{a.Domain()},
			Routes: []route.Route{{
				Match: route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: a.Domain(),
						},
					},
				},
			}},
		})
	}

	inst := &v2.RouteConfiguration{
		Name:         app.Domain(),
		VirtualHosts: vHosts,
	}

	return []cache.Resource{inst}, nil
}

func (rb *ResourceBuilder) BuildHTTPListener(app *Application) ([]cache.Resource, error) {
	rdsSource := rb.makeDataSource(internel.GetAfxMeta().PushMode)

	listenerName, routeName := app.Domain(), app.Domain()

	//accessLogConfig := &alg.HttpGrpcAccessLogConfig{
	//	CommonConfig: &alg.CommonGrpcAccessLogConfig{
	//		LogName: "access.log",
	//		GrpcService: &core.GrpcService{
	//			TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
	//				EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
	//					ClusterName: rb.xdsCluster,
	//				},
	//			},
	//		},
	//	},
	//}

	accessLogConfig := &alg.FileAccessLog{
		Path: "/dev/null",
		AccessLogFormat: &alg.FileAccessLog_Format{
			Format: accessFormat,
		},
	}

	accessTypeConfig, err := types.MarshalAny(accessLogConfig)
	if err != nil {
		return nil, err
	}

	_ = accessTypeConfig

	manager := &httpConnMgr.HttpConnectionManager{
		CodecType:  httpConnMgr.AUTO,
		StatPrefix: listenerName + "_stat",
		RouteSpecifier: &httpConnMgr.HttpConnectionManager_Rds{
			Rds: &httpConnMgr.Rds{
				ConfigSource:    *rdsSource,
				RouteConfigName: routeName,
			},
		},

		HttpFilters: []*httpConnMgr.HttpFilter{{
			Name: util.Router,
		}},

		//AccessLog: []*alf.AccessLog{{
		//	Name: util.HTTPGRPCAccessLog,
		//	ConfigType: &alf.AccessLog_TypedConfig{
		//		TypedConfig: accessTypeConfig,
		//	},
		//}},

		//AccessLog: []*alf.AccessLog{{
		//	Name: util.FileAccessLog,
		//	ConfigType: &alf.AccessLog_TypedConfig{
		//		TypedConfig: accessTypeConfig,
		//	},
		//}},

		GenerateRequestId: &types.BoolValue{Value: false}, // FIXME: disable x-request-id. not used trace so fa.
	}

	mgrTypeConfig, err := types.MarshalAny(manager)
	if err != nil {
		return nil, err
	}

	inst := &v2.Listener{
		Name: listenerName,
		Address: core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.TCP,
					Address:  listenAddress,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: listenPort,
					},
				},
			},
		},

		FilterChains: []listener.FilterChain{{
			Filters: []listener.Filter{{
				Name: util.HTTPConnectionManager,
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: mgrTypeConfig,
				},
			}},
		}},
	}

	return []cache.Resource{inst}, nil
}

func (rb *ResourceBuilder) makeDataSource(mode DataSourceMode) *core.ConfigSource {
	source := &core.ConfigSource{}

	switch mode {
	case AdsMode:
		source.ConfigSourceSpecifier = &core.ConfigSource_Ads{
			Ads: &core.AggregatedConfigSource{},
		}
	case XdsMode:
		source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &core.ApiConfigSource{
				ApiType: core.ApiConfigSource_GRPC,
				GrpcServices: []*core.GrpcService{{
					TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
							ClusterName: rb.xdsCluster,
						},
					},
				}},
			},
		}
	case RestMode:
		source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &core.ApiConfigSource{
				ApiType:      core.ApiConfigSource_REST,
				ClusterNames: []string{"not.implement.com"},
				RefreshDelay: &refreshDelay,
			},
		}
	}

	return source
}

func (rb *ResourceBuilder) BuildCustomAppEndpoints(app *Application) ([]cache.Resource, error) {

	var cfg []byte
	for _, res := range app.Resources {
		if res.ResType == ResourceTypeEndpoint {
			cfg = []byte(res.Config)
			break
		}
	}

	return rb.BuildCustomResource(app, ResourceTypeEndpoint, cfg)
}

func (rb *ResourceBuilder) BuildCustomResource(app *Application, resType ResourceType, cfg []byte) ([]cache.Resource, error) {
	tagLog := logs.FuncTaggedLoggerFactory()

	if resType == ResourceTypeEndpoint {
		ress, err := ValidationResource(resType, cfg)
		if err != nil {
			return nil, err
		}

		if err := StrictEndpointValidation(ress); err != nil {
			tagLog("strict check endpoints").Errorln(err)
			return rb.BuildFastAppEndpoints(app)
		}

		if err = app.completeXstream(); err != nil {
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
			if placeHoldLoadAssign.ClusterName == app.Hash() {
				continue
			}

			placeHoldEnds := placeHoldLoadAssign.Endpoints
			placeHoldLbe := placeHoldEnds[0].LbEndpoints[0]

			ends, ok := endsDict[placeHoldLoadAssign.ClusterName]
			if !ok {
				tagLog("lookup etcd upstream").Errorln("miss! cluster name:", placeHoldLoadAssign.ClusterName)
				continue
			}

			lbes := make([]endpoint.LbEndpoint, len(ends))
			for idx, end := range ends {
				/* LbEndpoint has pointer, must deep copy */
				lbe := deepcopy.Copy(placeHoldLbe).(endpoint.LbEndpoint)
				lbe.GetEndpoint().Address = &core.Address{
					Address: &core.Address_SocketAddress{
						SocketAddress: &core.SocketAddress{
							Protocol:      core.TCP,
							Address:       end.Address,
							PortSpecifier: &core.SocketAddress_PortValue{PortValue: end.Port},
						},
					},
				}

				lbes[idx] = lbe
			}
			placeHoldLoadAssign.Endpoints[0].LbEndpoints = lbes
		}

		return ress, err
	}

	return ValidationResource(resType, cfg)
}
