# Build the manager binary
FROM golang:1.16-buster AS builder

ARG VERSION=undefined

# Copy in the go src
WORKDIR /go/src/github.com/wave-k8s/wave

COPY go.mod go.mod
COPY go.sum go.sum

# Fetch dependencies before copying code (should cache unless go.mod, go.sum change)
RUN go mod download

COPY pkg/    pkg/
COPY cmd/    cmd/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o wave -ldflags="-X main.VERSION=${VERSION}" github.com/wave-k8s/wave/cmd/manager

# Copy the controller-manager into a thin image
FROM alpine:3.13
RUN apk --no-cache add ca-certificates
WORKDIR /bin
COPY --from=builder /go/src/github.com/wave-k8s/wave/wave .
ENTRYPOINT ["/bin/wave"]
