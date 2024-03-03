# Build the manager binary
FROM docker.io/golang:1.16 as builder

ARG VERSION=undefined

# Install Dep
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

# Copy in the go src
WORKDIR /go/src/github.com/wave-k8s/wave

# Get the dep requirements in
COPY go.mod .
COPY go.sum .

# Download deps
RUN go mod download -x

# Put .go source into the container
COPY cmd ./cmd
COPY pkg ./pkg
COPY test ./test

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -a -o wave -ldflags="-X main.VERSION=${VERSION}" github.com/wave-k8s/wave/cmd/manager

# Copy the controller-manager into a thin image
FROM docker.io/alpine:3.14
RUN apk --no-cache add ca-certificates
WORKDIR /bin
COPY --from=builder /go/src/github.com/wave-k8s/wave/wave .
ENTRYPOINT ["/bin/wave"]
