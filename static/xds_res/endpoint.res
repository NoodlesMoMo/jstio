
{{- range $protocol := .Protocols }}
- cluster_name: {{ $protocol.Domain }}
  endpoints:
    - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 127.0.0.1
                port_value: {{ $protocol.AppPort }}
  {{- range $upstream := $.Upstream }}
    {{- range $upstreamProto := $upstream.Protocols }}
      {{- if eq $upstreamProto.Protocol $protocol.Protocol }}
- cluster_name: {{ $upstreamProto.Domain }}
  endpoints:
    - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 127.0.0.1 #dummy address
                port_value: {{ $upstreamProto.ProxyPort }}
      {{- end }}
    {{- end}}
  {{- end }}
{{- end }}
