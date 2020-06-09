package model

import (
	"fmt"
	"testing"
)

const (
	testListener = `[
    {
        "address":{
            "socket_address":{
                "address":"0.0.0.0",
                "port_value":80
            }
        },
        "filter_chains":[
            {
                "filters":[
                    {
                        "name":"envoy.http_connection_manager",
                        "typed_config":{
                            "@type":"type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
                            "generate_request_id":false,
                            "http_filters":[
                                {
                                    "name":"envoy.router"
                                }
                            ],
                            "rds":{
                                "config_source":{
                                    "ads":{

                                    }
                                },
                                "route_config_name":"envoy-backend.test.odin.sogou"
                            },
                            "stat_prefix":"envoy-backend.test.odin.sogou_stat"
                        }
                    }
                ]
            }
        ],
        "name":"envoy-backend.test.odin.sogou"
    },
    {
        "address":{
            "socket_address":{
                "address":"0.0.0.0",
                "port_value":3721
            }
        },
        "filter_chains":[
            {
                "filters":[
                    {
                        "name":"envoy.http_connection_manager",
                        "typed_config":{
                            "@type":"type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
                            "generate_request_id":false,
                            "http_filters":[
                                {
                                    "name":"envoy.router"
                                }
                            ],
                            "rds":{
                                "config_source":{
                                    "ads":{

                                    }
                                },
                                "route_config_name":"envoy-backend.grpc.test.odin.sogou"
                            },
                            "stat_prefix":"envoy-backend.grpc.test.odin.sogou_stat"
                        }
                    }
                ]
            }
        ],
        "name":"envoy-backend.grpc.test.odin.sogou"
    }
]`

	testCluster = `[
    {
        "name":"envoy-backend.test.odin.sogou",
        "type":"EDS",
        "circuit_breakers":{
            "thresholds":[
                {
                    "max_connections":1000000000,
                    "max_pending_requests":1000000000,
                    "max_requests":1000000000,
                    "max_retries":3,
                    "priority":"HIGH"
                }
            ]
        },
        "connect_timeout":"3s",
        "eds_cluster_config":{
            "eds_config":{
                "ads":{

                }
            }
        }
    },
    {
        "name":"envoy-backend.grpc.test.odin.sogou",
        "type":"EDS",
        "circuit_breakers":{
            "thresholds":[
                {
                    "max_connections":1000000000,
                    "max_pending_requests":1000000000,
                    "max_requests":1000000000,
                    "max_retries":3,
                    "priority":"HIGH"
                }
            ]
        },
        "connect_timeout":"3s",
        "http2_protocol_options":{

        },
        "eds_cluster_config":{
            "eds_config":{
                "ads":{

                }
            }
        }
    }
]`
)

func TestListenerRess(t *testing.T) {
	ress, err := ValidationResource(ResourceTypeListener, []byte(testListener))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+v\n", ress)
	//_, _ = pp.Println(ress)

	ress2, err := ValidationResourceV2(ResourceTypeListener, []byte(testListener))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+v\n", ress2)
	//_, _ = pp.Println(ress2)
}

func TestClusterRess(t *testing.T) {
	ress, err := ValidationResource(ResourceTypeCluster, []byte(testCluster))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+v\n", ress)
	//_, _ = pp.Println(ress)

	ress2, err := ValidationResourceV2(ResourceTypeCluster, []byte(testCluster))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+v\n", ress2)
	//_, _ = pp.Println(ress2)
}
