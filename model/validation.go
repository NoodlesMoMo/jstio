package model

import (
	"bytes"
	"encoding/json"
	"errors"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
)

var (
	ErrValidateType             = errors.New("resource validate: unknown type")
	ErrValidateEndpointName     = errors.New("resource validate: empty endpoint name")
	ErrValidateEndpointNullLLBE = errors.New("resource validate: empty endpoint")
)

type validateResource interface {
	proto.Message

	Validate() error
	Equal(interface{}) bool
}

func resourceMakerFactory(resType ResourceType) func() validateResource {
	switch resType {
	case ResourceTypeRoute:
		return func() validateResource {
			return new(v2.RouteConfiguration)
		}
	case ResourceTypeListener:
		return func() validateResource {
			return new(v2.Listener)
		}
	case ResourceTypeCluster:
		return func() validateResource {
			return new(v2.Cluster)
		}
	case ResourceTypeEndpoint:
		return func() validateResource {
			return new(v2.ClusterLoadAssignment)
		}
	}

	panic(ErrValidateType)
}

func ValidationResource(resType ResourceType, content []byte) ([]cache.Resource, error) {
	var (
		err error
	)

	isArray := false
	rcMaker := resourceMakerFactory(resType)

	switch resType {
	case ResourceTypeEndpoint, ResourceTypeCluster:
		isArray = true
	}

	if isArray {
		var clusters []map[string]interface{}
		if err = json.Unmarshal(content, &clusters); err != nil {
			return nil, err
		}

		var ress []cache.Resource
		for _, cluster := range clusters {
			b, _ := json.Marshal(cluster)
			rc := rcMaker()
			if err = jsonpb.Unmarshal(bytes.NewReader(b), rc); err != nil {
				return nil, err
			}
			ress = append(ress, rc)
			if e := rc.Validate(); e != nil {
				return nil, e
			}
		}
		return ress, nil
	}

	res := rcMaker()

	if err = jsonpb.Unmarshal(bytes.NewReader(content), res); err != nil {
		return nil, err
	}

	return []cache.Resource{res}, res.Validate()
}

func StrictEndpointValidation(ress []cache.Resource) error {
	if ress == nil {
		return errors.New("invalid param")
	}

	for _, res := range ress {
		cla, ok := res.(*v2.ClusterLoadAssignment)
		if !ok {
			return ErrValidateType
		}

		if cla.ClusterName == "" {
			return ErrValidateEndpointName
		}

		if len(cla.Endpoints) == 0 || len(cla.Endpoints[0].LbEndpoints) == 0 {
			return ErrValidateEndpointNullLLBE
		}
	}

	return nil
}
