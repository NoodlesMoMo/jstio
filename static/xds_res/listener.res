address:
  socket_address:
    address: 0.0.0.0
    port_value: 80
filter_chains:
  - filters:
      - name: envoy.http_connection_manager
        typed_config:
          '@type': type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager
          generate_request_id: false
          http_filters:
            - name: envoy.router
          rds:
            config_source:
              ads: {}
            route_config_name: {{ .Hash }}
          stat_prefix: {{ .Hash }}_stat
name: {{ .Hash }}
