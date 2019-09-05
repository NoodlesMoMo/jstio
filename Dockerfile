FROM golang:1.12-alpine3.9 AS build

RUN mkdir -p /gogo/bin

RUN apk add --no-cache git

ADD . /gogo

WORKDIR /gogo

RUN go build -o /gogo/bin/jstio

FROM alpine:3.9

RUN apk add --no-cache bash

COPY --from=build /gogo/bin /bin
COPY --from=build /gogo/conf /bin/conf

EXPOSE 8000 8080

WORKDIR /bin

CMD ["jstio"]

