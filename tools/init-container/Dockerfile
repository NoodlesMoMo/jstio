FROM golang:1.12 as build

RUN mkdir -p /gogo/src/envoy-init && \
    mkdir -p /tmp/build && \
    mkdir -p /tmp/build/template

COPY . /gogo/src/envoy-init

WORKDIR /gogo/src/envoy-init

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/build/envoy-init && \
    cp -rf * /tmp/build

#************************************#

FROM alpine

RUN mkdir -p /sogou/bin /sogou/bin/template

RUN apk update && apk add --no-cache bash

COPY --from=build /tmp/build /sogou/bin

WORKDIR /sogou/bin

ENTRYPOINT ["./docker-entrypoint.sh"]
CMD ["envoy-init"]
