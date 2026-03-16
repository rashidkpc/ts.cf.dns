FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o ts-cf-dns .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/ts-cf-dns /usr/local/bin/ts-cf-dns
ENTRYPOINT ["ts-cf-dns"]
