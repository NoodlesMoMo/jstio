
## ElasticSearch deployment

### index by day
    index pattern  like this: `envoy_accesslog_20191201`

### index template

    PUT _template/envoy_accesslog
    {
      "index_patterns": ["envoy_accesslog*"],
      "settings": {
        "number_of_shards": 5
      },
      "mappings": {
        "properties": {
          "app": {"type":"keyword"},
          "pod": {"type":"keyword"},
          "oc": {"type":"keyword"},
          "domain":{"type":"keyword"},
          "level": {"type": "keyword"},
          "file_name": {"type": "keyword"},
          "req_id": {"type":"keyword"},
          "upstream_addr": {"type": "ip"},
          "upstream_cluster":{"type":"keyword"},
          "upstream_fail_reason":{"type":"keyword"},
          "star_time":{"type": "date"},
          "elapsed": {"type": "double"},
          "protocol": {"type":"keyword"},
          "req_method": {"type": "keyword"},
          "authority": {"type":"keyword"},
          "req_header_len": {"type":"long"},
          "resp_code": {"type":"long"},
          "resp_header_len":{"type":"long"},
          "resp_body_len": {"type":"long"}
        }
      }
    }

### envoy_accesslog agg query

    GET /envoy_accesslog_test/_search
    {
      "size": 0,
      "aggs": {
        "group_by_domain": {
          "terms": {
            "field": "domain"
          },
          
          "aggs": {
            "group_by_level": {
              "terms": {
                "field": "level"
              }
            }
          }
        }
      }
    }
