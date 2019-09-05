- circuit_breakers:
    thresholds:
      - max_connections: 1000000000
        max_pending_requests: 1000000000
        max_requests: 1000000000
        max_retries: 3
        priority: HIGH
  connect_timeout: 3s
  eds_cluster_config:
    eds_config:
      ads: {}
  name: {{ .Hash }}
  type: EDS
{{ range $app := .Upstream }}
  {{ template "cluster" $app}}
{{ end }}
