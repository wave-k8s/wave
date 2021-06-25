# Build the manager binary
FROM golang:1.16 as builder

ARG VERSION=undefined

# Install Dep
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

# Copy in the go src
WORKDIR /go/src/github.com/wave-k8s/wave
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -a -o wave -ldflags="-X main.VERSION=${VERSION}" github.com/wave-k8s/wave/cmd/manager

# Copy the controller-manager into a thin image
FROM alpine:3.14
RUN apk --no-cache add ca-certificates
WORKDIR /bin
COPY --from=builder /go/src/github.com/wave-k8s/wave/wave .
ENTRYPOINT ["/bin/wave"]
