FROM golang:1.23-alpine

WORKDIR /go

RUN apk add --no-cache git ca-certificates

ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$PATH