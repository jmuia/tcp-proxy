FROM golang:latest as builder

COPY . $GOPATH/src/github.com/jmuia/tcp-proxy/
WORKDIR $GOPATH/src/github.com/jmuia/tcp-proxy/

RUN go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -o /usr/local/bin/tcp-proxy

FROM scratch
COPY --from=builder /usr/local/bin/tcp-proxy /usr/local/bin/tcp-proxy

ENTRYPOINT ["/usr/local/bin/tcp-proxy"]

