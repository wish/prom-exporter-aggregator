FROM golang:1.12
COPY . /go/src/github.com/wish/prom-exporter-aggregator/
ENV GO111MODULE=on
WORKDIR /go/src/github.com/wish/prom-exporter-aggregator/cmd/prom
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .



FROM alpine:3.7
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/wish/prom-exporter-aggregator/cmd/prom/prom .
CMD /root/prom
