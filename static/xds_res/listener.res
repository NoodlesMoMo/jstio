
{{- range $protocol := .Protocols }}
- name: {{ $protocol.Domain }}
  address:
    socket_address:
      address: 0.0.0.0
      port_value: {{ $protocol.ProxyPort }}
  filter_chains:
    - filters:
      - name: envoy.http_connection_manager
        typed_config:
          '@type': type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager
          http_filters:
            - name: envoy.router
          generate_request_id: false
          {{- if eq $protocol.Protocol "http" }}
          tracing:
            operation_name: EGRESS
            random_sampling:
              value: 0.001
          access_log:
            - name: envoy.http_grpc_access_log
              filter:
                status_code_filter:
                  comparison:
                    op: GE
                    value:
                      runtime_key: http_ge_400
                      default_value: 400
              config:
                common_config:
                  log_name: non200/non200
                  grpc_service:
                    envoy_grpc:
                      cluster_name: xds_logserver
            - name: envoy.http_grpc_access_log
              filter:
                duration_filter:
                  comparison:
                    op: GE
                    value:
                      runtime_key: timeout
                      default_value: 200
              config:
                common_config:
                  log_name: timeout/timeout
                  grpc_service:
                    envoy_grpc:
                      cluster_name: xds_logserver
            {{- end }}
          rds:
            config_source:
              ads: {}
            route_config_name: {{ $protocol.Domain }}
          stat_prefix: {{$protocol.Domain}}_stat
{{- end }}
