FROM golang:1.21 AS builder

RUN apt update -y && \
  apt install -y ca-certificates

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o prenv .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/prenv /usr/local/bin/prenv
