FROM golang:1.15-alpine AS builder

WORKDIR /go/src/app
RUN apk update && apk add --no-cache git ca-certificates
COPY . .
RUN GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/instrumentation-dcs-loader loader/main.go

FROM scratch
COPY --from=builder /go/src/app/bin/instrumentation-dcs-loader /go/bin/instrumentation-dcs-loader
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
CMD ["/go/bin/instrumentation-dcs-loader"]