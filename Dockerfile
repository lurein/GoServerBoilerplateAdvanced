# Build the docker image
FROM alpine:3.14 as image
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
RUN update-ca-certificates

# Build the go binary
FROM golang:1.17.3 AS build

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Move to working directory /src
WORKDIR /src

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Build the application
RUN go build -o /bin/whimsy .

# Add the image dependencies
FROM image

# Copy binary
COPY --from=build /bin/sycamore /bin/whimsy
CMD /bin/whimsy server
