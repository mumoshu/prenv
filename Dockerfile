FROM golang:1.21

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o prenv .

FROM scratch

COPY --from=0 /app/prenv /usr/local/bin/prenv
