FROM golang:1.18-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/inaply-gateway ./cmd/gateway

FROM alpine:3.19
RUN apk add --no-cache nftables ca-certificates
COPY --from=builder /out/inaply-gateway /usr/local/bin/inaply-gateway
EXPOSE 8080/tcp 5353/udp 1900/udp
ENTRYPOINT ["/usr/local/bin/inaply-gateway"]
