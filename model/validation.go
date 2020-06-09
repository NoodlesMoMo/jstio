package model

import (
	"bytes"
	"encoding/json"
	"errors"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

var (
	ErrValidateType = errors.New("resource validate: unknown type")
)

type validateResource interface {
	proto.Message
	Validate() error
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
	var err error

	rcMaker := resourceMakerFactory(resType)

	var materials []map[string]interface{}
	if err = json.Unmarshal(content, &materials); err != nil {
		return nil, err
	}

	var cacheRess []cache.Resource
	for _, material := range materials {
		b, _ := json.Marshal(material)
		rc := rcMaker()
		if err = jsonpb.Unmarshal(bytes.NewReader(b), rc); err != nil {
			return nil, err
		}
		if err = rc.Validate(); err != nil {
			return nil, err
		}
		cacheRess = append(cacheRess, rc)
	}
	return cacheRess, nil
}

func ValidationResourceV2(resType ResourceType, content []byte) ([]cache.Resource, error) {
	var err error

	rcMaker := resourceMakerFactory(resType)

	jsonDecoder := json.NewDecoder(bytes.NewReader(content))
	_, err = jsonDecoder.Token()

	var cacheRess []cache.Resource
	for jsonDecoder.More() {
		rc := rcMaker()
		if err = jsonpb.UnmarshalNext(jsonDecoder, rc); err != nil {
			return nil, err
		}
		if err = rc.Validate(); err != nil {
			return nil, err
		}
		cacheRess = append(cacheRess, rc)
	}
	return cacheRess, nil
}
