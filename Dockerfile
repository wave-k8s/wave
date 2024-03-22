# Build the manager binary
FROM golang:1.22 as builder

ARG VERSION=undefined

# Set the working directory
WORKDIR /go/src/github.com/wave-k8s/wave

# Copy the go mod and sum files
COPY go.mod go.sum ./

# Fetch dependencies, leveraging Docker cache layers
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o wave -ldflags="-X main.VERSION=${VERSION}" ./cmd/manager

# Copy the controller-manager into a thin image
FROM alpine:3.11
RUN apk --no-cache add ca-certificates
WORKDIR /bin
COPY --from=builder /go/src/github.com/wave-k8s/wave/wave .
ENTRYPOINT ["/bin/wave"]
