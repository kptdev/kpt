FROM golang:1.13-stretch

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go build -v -o /bin/app .

CMD ["app"]