{{- range $selfProto := $.Protocols }}
- name: {{ $selfProto.Domain }}
  type: EDS
  circuit_breakers:
    thresholds:
      - max_connections: 1000000000
        max_pending_requests: 1000000000
        max_requests: 1000000000
        max_connection_pools: 1000000000
        max_retries: 3
  connect_timeout: 3s
  {{- if eq $selfProto.Protocol "grpc" }}
  http2_protocol_options: {}
  {{- end }}
  eds_cluster_config:
    eds_config:
      ads: {}
  {{- range $upstream := $.Upstream }}
    {{- range $upstreamProto := $upstream.Protocols }}
      {{- if eq $upstreamProto.Protocol $selfProto.Protocol }}
- name: {{ $upstreamProto.Domain }}
  type: EDS
  circuit_breakers:
    thresholds:
      - max_connections: 1000000000
        max_pending_requests: 1000000000
        max_requests: 1000000000
        max_connection_pools: 1000000000
        max_retries: 3
  connect_timeout: 3s
  {{- if eq $upstreamProto.Protocol "grpc" }}
  http2_protocol_options: {}
  {{- end }}
  eds_cluster_config:
    eds_config:
      ads: {}
      {{- end }}
    {{- end }}
  {{- end}}
{{- end }}
