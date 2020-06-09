{{- range $selfProtocol := .Protocols }}
- name: {{ $selfProtocol.Domain }}
  virtual_hosts:
    - domains:
      - {{ $selfProtocol.Domain }}
      name: {{ $selfProtocol.Domain }}
      routes:
        - match:
            prefix: /
          route:
            timeout: 2s
            cluster: {{ $selfProtocol.Domain }}
  {{- range $upstream := $.Upstream }}
    {{- range $upstreamProtocol := $upstream.Protocols }}
      {{- if eq $upstreamProtocol.Protocol $selfProtocol.Protocol }}
    - domains:
      - {{ $upstreamProtocol.Domain }}
      name: {{ $upstreamProtocol.Domain  }}
      routes:
        - match:
            prefix: /
          route:
            timeout: 2s
            cluster: {{ $upstreamProtocol.Domain }}
      {{- end }}
    {{- end }}
  {{- end }}
{{- end }}
