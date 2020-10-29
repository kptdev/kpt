# Copyright 2019 Google Inc.  All Rights Reserved.

FROM golang:1.13-stretch
ENV CGO_ENABLED=0
WORKDIR /go/src/
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
WORKDIR functions/examples
RUN go build -v -o /usr/local/bin/config-function

FROM alpine:latest
COPY --from=0 /usr/local/bin/config-function /usr/local/bin/config-function
CMD ["config-function"]
