FROM golang:1.21 AS builder

RUN apt update -y && \
  apt install -y ca-certificates

# WORKDIR /app

# COPY go.mod ./
# COPY go.sum ./

# RUN go mod download

# COPY . .

# RUN CGO_ENABLED=0 go build -o prenv .

FROM ubuntu:jammy as deps

ARG TARGETOS
ARG TARGETARCH
ARG KUBECTL_VERSION=1.28.2

ADD https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/${TARGETOS}/${TARGETARCH}/kubectl /tmp
RUN mv /tmp/kubectl /usr/local/bin/kubectl \
  && chmod 755 /usr/local/bin/kubectl

FROM amazon/aws-cli:2.13.33

COPY --from=deps /usr/local/bin/kubectl /usr/local/bin/kubectl
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# COPY --from=builder /app/prenv /usr/local/bin/prenv
COPY prenv /usr/local/bin/prenv

ENTRYPOINT ["/usr/local/bin/prenv"]
