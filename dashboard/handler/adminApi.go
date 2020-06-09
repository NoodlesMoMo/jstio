package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"git.sogou-inc.com/iweb/jstio/dashboard/service"
	"git.sogou-inc.com/iweb/jstio/model"
	envoyApiV2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoyApiV2Route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"
	routing "github.com/qiangxue/fasthttp-routing"
)

type RouteConfiguration envoyApiV2.RouteConfiguration

func (r *RouteConfiguration) UnmarshalJSON(b []byte) error {
	cfg := envoyApiV2.RouteConfiguration{}
	err := jsonpb.Unmarshal(bytes.NewReader(b), &cfg)
	*r = RouteConfiguration(cfg)
	return err
}

func (r RouteConfiguration) MarshalJSON() ([]byte, error) {
	m := jsonpb.Marshaler{}
	cfg := envoyApiV2.RouteConfiguration(r)
	b, err := m.MarshalToString(&cfg)
	return []byte(b), err
}

type Route envoyApiV2Route.Route

func (r *Route) UnmarshalJSON(b []byte) error {
	route := envoyApiV2Route.Route{}
	err := jsonpb.Unmarshal(bytes.NewReader(b), &route)
	*r = Route(route)
	return err
}

func (r Route) MarshalJSON() ([]byte, error) {
	m := jsonpb.Marshaler{}
	route := envoyApiV2Route.Route(r)
	b, err := m.MarshalToString(&route)
	return []byte(b), err
}

type Cluster envoyApiV2.Cluster

func (c *Cluster) UnmarshalJSON(b []byte) error {
	cluster := envoyApiV2.Cluster{}
	err := jsonpb.Unmarshal(bytes.NewReader(b), &cluster)
	*c = Cluster(cluster)
	return err
}

func (c Cluster) MarshalJSON() ([]byte, error) {
	m := jsonpb.Marshaler{}
	cluster := envoyApiV2.Cluster(c)
	b, err := m.MarshalToString(&cluster)
	return []byte(b), err
}

func GetRoutes(ctx *routing.Context) error {
	appName := string(ctx.QueryArgs().Peek("app_name"))
	cluster := string(ctx.QueryArgs().Peek("cluster"))

	if appName == "" {
		return service.ErrorResponse(ctx, -1, "app_name empty")
	}

	if cluster == "" {
		return service.ErrorResponse(ctx, -1, "cluster empty")
	}

	routeRes, err := routeResource(appName, cluster, model.ResourceTypeRoute)
	if err != nil {
		return service.ErrorResponse(ctx, -2, "no route resources")
	}

	routeConfigs := []RouteConfiguration{}
	err = json.Unmarshal([]byte(routeRes.Config), &routeConfigs)
	if err != nil {
		return service.ErrorResponse(ctx, -2, "json unmasharl fail")
	}

	routeMap := map[string]map[string][]Route{}
	for _, routeConfig := range routeConfigs {
		cfg := map[string][]Route{}
		for _, vhost := range routeConfig.VirtualHosts {
			vRoutes := []Route{}
			for _, route := range vhost.Routes {
				vRoutes = append(vRoutes, Route(*route))
			}
			cfg[vhost.Name] = vRoutes
		}
		routeMap[routeConfig.Name] = cfg
	}

	return service.SuccessResponse(ctx, routeMap)
}

func UpdateRoutes(ctx *routing.Context) error {
	routeCfg := string(ctx.QueryArgs().Peek("route_config"))
	appName := string(ctx.QueryArgs().Peek("app_name"))
	cluster := string(ctx.QueryArgs().Peek("cluster"))

	if routeCfg == "" {
		return service.ErrorResponse(ctx, -1, "empty route config")
	}
	if appName == "" {
		return service.ErrorResponse(ctx, -1, "empty app name")
	}
	if cluster == "" {
		return service.ErrorResponse(ctx, -1, "empty cluster")
	}

	cfg := map[string]map[string][]Route{}
	err := json.Unmarshal([]byte(routeCfg), &cfg)
	if err != nil {
		return service.ErrorResponse(ctx, -1, "param route unmashal err")
	}

	routeRes, err := routeResource(appName, cluster, model.ResourceTypeRoute)
	if err != nil {
		return service.ErrorResponse(ctx, -2, "no route resource")
	}

	routeConfigs := []RouteConfiguration{}
	err = json.Unmarshal([]byte(routeRes.Config), &routeConfigs)
	if err != nil {
		return service.ErrorResponse(ctx, -2, "db config ummashal fail")
	}

	for rIdx, routeConfig := range routeConfigs {
		for vIdx, vhost := range routeConfig.VirtualHosts {
			routes := []*envoyApiV2Route.Route{}

			pVhost, ok := cfg[routeConfig.Name]
			if !ok {
				continue
			}

			pRoutes, ok := pVhost[vhost.Name]
			if !ok || len(pRoutes) == 0 {
				continue
			}

			for _, pRoute := range pRoutes {
				route := envoyApiV2Route.Route(pRoute)
				routes = append(routes, &route)
			}

			routeConfigs[rIdx].VirtualHosts[vIdx].Routes = routes
		}
	}

	jsonConfig, err := json.Marshal(routeConfigs)
	if err != nil {
		return service.ErrorResponse(ctx, -2, "config mashal fail")
	}
	yamlConfig, err := yaml.JSONToYAML(jsonConfig)
	if err != nil {
		return service.ErrorResponse(ctx, -2, "json to yaml fail")
	}
	routeRes.Config = string(jsonConfig)
	routeRes.YamlConfig = string(yamlConfig)

	err = routeRes.Update(true)
	if err != nil {
		return service.ErrorResponse(ctx, -2, "update db fail")
	}

	return service.SuccessResponse(ctx, "")
}

func GetClusterList(ctx *routing.Context) error {
	appName := string(ctx.QueryArgs().Peek("app_name"))
	cluster := string(ctx.QueryArgs().Peek("cluster"))

	if appName == "" {
		return service.ErrorResponse(ctx, -1, "no app name")
	}

	if cluster == "" {
		return service.ErrorResponse(ctx, -1, "no cluster")
	}

	clusterRes, err := routeResource(appName, cluster, model.ResourceTypeCluster)
	if err != nil {
		return service.ErrorResponse(ctx, -2, err)
	}

	clusters := []Cluster{}
	err = json.Unmarshal([]byte(clusterRes.Config), &clusters)
	if err != nil {
		return service.ErrorResponse(ctx, -2, err)
	}

	clusterNames := []string{}
	for _, cluster := range clusters {
		clusterNames = append(clusterNames, cluster.Name)
	}

	return service.SuccessResponse(ctx, clusterNames)
}

func routeResource(appName string, cluster string, resType model.ResourceType) (model.Resource, error) {
	resource := model.Resource{}
	app, err := model.GetSingleAppByName(appName, cluster)
	if err != nil {
		return resource, err
	}

	routeExist := false
	for _, res := range app.Resources {
		if res.ResType == resType {
			resource = res
			routeExist = true
			break
		}
	}

	if !routeExist {
		return resource, errors.New("no route resource")
	}

	return resource, nil
}
