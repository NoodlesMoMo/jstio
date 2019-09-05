name: {{ .Hash }}
virtual_hosts:
  - domains:
      - {{ .Hash }}
    include_request_attempt_count: true
    name: {{ .Hash }}
    retry_policy:
      num_retries: 3
      per_try_timeout: 2s
      retry_on: reset,connect-failure,refused-stream,gateway-error
      retry_host_predicate:
        - name: envoy.retry_host_predicates.previous_hosts
      host_selection_retry_max_attempts: 3
      retry_priority:
        name: envoy.retry_priorities.previous_priorities
        config:
          update_frequency: 2
    routes:
      - match:
          prefix: /
        route:
          cluster: {{ .Hash }}
{{ range $app := .Upstream }}
  {{ template "route" $app }}
{{ end }}
