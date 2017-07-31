# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.8

MAINTAINER Kitae Kim

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/superkkt/cherry/
COPY ./cherry.yaml /usr/local/etc/
 
# Build cherry inside the container.
RUN go install github.com/superkkt/cherry

# Run the cherry command by default when the container starts.
ENTRYPOINT ["/go/bin/cherry"]

# Document that the service listens on port 6633.
EXPOSE 6633
