FROM golang:1.22-alpine as builder
RUN apk add --update \
    make \
    git \
    && rm -rf /var/cache/apk/*
RUN mkdir -p /go/src/github.com/mheers/trino-gateway-operator
ADD . /go/src/github.com/mheers/trino-gateway-operator
WORKDIR /go/src/github.com/mheers/trino-gateway-operator
RUN make build

FROM alpine:3.19

COPY --from=builder /go/src/github.com/mheers/trino-gateway-operator/bin/trino-gateway-operator /trino-gateway-operator
ENTRYPOINT ["/trino-gateway-operator"]
