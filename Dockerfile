FROM golang:1.21 AS builder
# Set necessary environment variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o mysql-backup main.go


FROM alpine

WORKDIR /app
COPY --from=builder /app .
RUN apt-get update && apt-get install -y git curl libmcrypt-dev default-mysql-client
RUN rm -rf /var/cache/apk/*

CMD ["./mysql-backup"]
