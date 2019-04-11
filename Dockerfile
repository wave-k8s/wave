# Build the manager binary
FROM golang:1.12 as builder

ARG VERSION=undefined

# Install Dep
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

# Copy in the go src
WORKDIR /go/src/github.com/pusher/wave
COPY Gopkg.lock Gopkg.lock
COPY Gopkg.toml Gopkg.toml

# Fetch dependencies before copying code (should cache unless Gopkg's change)
RUN dep ensure --vendor-only

COPY pkg/    pkg/
COPY cmd/    cmd/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o wave -ldflags="-X main.VERSION=${VERSION}" github.com/pusher/wave/cmd/manager

# Copy the controller-manager into a thin image
FROM alpine:3.9
RUN apk --no-cache add ca-certificates
WORKDIR /bin
COPY --from=builder /go/src/github.com/pusher/wave/wave .
ENTRYPOINT ["/bin/wave"]
