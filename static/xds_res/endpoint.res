- cluster_name: {{ .Hash }}
  endpoints:
    - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 127.0.0.1
                port_value: {{ .AppPorts }}
{{ range $app := .Upstream }}
  {{ template "endpoint" $app }}
{{ end }}
