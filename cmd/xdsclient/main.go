package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/pkg/conversion"
	"github.com/golang/protobuf/ptypes"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"log"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoylistener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	envoyhttp "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/k0kubun/pp"
	"google.golang.org/grpc"
)

var dr envoy_api_v2.DiscoveryRequest

var marshaler = &jsonpb.Marshaler{OrigName: true}

func init() {
	dr.Node = new(envoy_api_v2_core.Node)
}

func GetYaml(pb proto.Message) []byte {
	buf := &bytes.Buffer{}

	err := marshaler.Marshal(buf, pb)
	if err != nil {
		panic(err)
	}
	data, err := yaml.JSONToYAML(buf.Bytes())
	if err != nil {
		panic(err)
	}
	return data
}

func main() {
	appName := flag.String("app", "srv-dict", "app name")
	cluster := flag.String("cluster", "venus", "odin cluster")
	port := flag.String("port", "8080", "xds manager port")
	host := flag.String("host", "jstio.shouji.sogou", "xds manager host")

	flag.Parse()

	dr.Node.Cluster = *appName
	dr.Node.Metadata = &structpb.Struct{
		Fields: map[string]*structpb.Value{"odin_cluster": {Kind: &structpb.Value_StringValue{StringValue: *cluster}}},
	}

	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", *host, *port), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("dial err: %v", err)
	}
	ctx := context.Background()
	clusters := listClusters(ctx, conn)
	listeners := listListeners(ctx, conn)
	var hcms []envoyhttp.HttpConnectionManager
	for _, l := range listeners {

		for _, fc := range l.FilterChains {
			for _, filter := range fc.Filters {
				if filter.Name == "envoy.http_connection_manager" {
					var hcm envoyhttp.HttpConnectionManager
					switch config := filter.ConfigType.(type) {
					case *envoylistener.Filter_Config:
						//if err := envoyutil.StructToMessage(config.Config, &hcm); err == nil {
						if err := conversion.StructToMessage(config.Config, &hcm); err == nil {
							hcms = append(hcms, hcm)
						}
					case *envoylistener.Filter_TypedConfig:
						//if err := types.UnmarshalAny(config.TypedConfig, &hcm); err == nil {
						if err := ptypes.UnmarshalAny(config.TypedConfig, &hcm); err == nil {
							hcms = append(hcms, hcm)
						}
					}
				}
			}
		}
	}

	var routes []string
	for _, hcm := range hcms {
		routes = append(routes, hcm.RouteSpecifier.(*envoyhttp.HttpConnectionManager_Rds).Rds.RouteConfigName)
	}

	rds := listRoutes(ctx, conn, routes)

	eds := listEndpoints(ctx, conn)

	fmt.Printf("\n\n#clusters")
	for _, c := range clusters {
		fmt.Printf("\n%s", GetYaml(&c))
	}
	fmt.Printf("\n\n#listeners")
	for _, c := range listeners {
		fmt.Printf("\n%s", GetYaml(&c))
	}
	fmt.Printf("\n\n#rds")
	for _, c := range rds {
		fmt.Printf("\n%s", GetYaml(&c))
	}
	fmt.Printf("\n\n#eds")
	for _, c := range eds {
		fmt.Printf("\n%s", GetYaml(&c))
	}
}

func listClusters(ctx context.Context, conn *grpc.ClientConn) []envoy_api_v2.Cluster {

	// clusters
	cdsc := envoy_api_v2.NewClusterDiscoveryServiceClient(conn)
	dresp, err := cdsc.FetchClusters(ctx, &dr)
	if err != nil {
		log.Fatalf("clusters err: %v", err)
	}
	var clusters []envoy_api_v2.Cluster
	for _, anyCluster := range dresp.Resources {
		var cluster envoy_api_v2.Cluster
		//cluster.Unmarshal(anyCluster.Value)
		ptypes.UnmarshalAny(anyCluster, &cluster)
		clusters = append(clusters, cluster)
		pp.Printf("%v\n", cluster)
	}
	return clusters
}

func listEndpoints(ctx context.Context, conn *grpc.ClientConn) []envoy_api_v2.ClusterLoadAssignment {
	eds := envoy_api_v2.NewEndpointDiscoveryServiceClient(conn)
	dresp, err := eds.FetchEndpoints(ctx, &dr)
	if err != nil {
		log.Fatalf("endpoints err: %v", err)
	}
	var class []envoy_api_v2.ClusterLoadAssignment
	pp.Printf("version info: %v\n", dresp.VersionInfo)

	for _, anyCla := range dresp.Resources {

		var cla envoy_api_v2.ClusterLoadAssignment
		//cla.Unmarshal(anyCla.Value)
		ptypes.UnmarshalAny(anyCla, &cla)
		pp.Printf("%v\n", cla)
		class = append(class, cla)
	}
	return class
}

func listListeners(ctx context.Context, conn *grpc.ClientConn) []envoy_api_v2.Listener {

	// listeners
	ldsc := envoy_api_v2.NewListenerDiscoveryServiceClient(conn)
	dresp, err := ldsc.FetchListeners(ctx, &dr)
	if err != nil {
		log.Fatalf("listeners err: %v", err)
	}
	var listeners []envoy_api_v2.Listener

	for _, anylistener := range dresp.Resources {
		var listener envoy_api_v2.Listener
		//listener.Unmarshal(anylistener.Value)
		ptypes.UnmarshalAny(anylistener, &listener)
		pp.Printf("%v\n", listener)
		listeners = append(listeners, listener)
	}
	return listeners
}

func listRoutes(ctx context.Context, conn *grpc.ClientConn, routenames []string) []envoy_api_v2.RouteConfiguration {

	// listeners
	ldsc := envoy_api_v2.NewRouteDiscoveryServiceClient(conn)

	drr := dr

	drr.ResourceNames = routenames

	dresp, err := ldsc.FetchRoutes(ctx, &drr)
	if err != nil {
		log.Fatalf("routes err: %v", err)
	}
	var routes []envoy_api_v2.RouteConfiguration

	for _, anyRoute := range dresp.Resources {
		var route envoy_api_v2.RouteConfiguration
		//route.Unmarshal(anyRoute.Value)
		ptypes.UnmarshalAny(anyRoute, &route)
		pp.Printf("%v\n", route)
		routes = append(routes, route)
	}
	return routes
}
