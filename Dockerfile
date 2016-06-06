# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

MAINTAINER Kitae Kim

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/superkkt/cherry/
COPY ./cherryd/cherryd.conf /usr/local/etc/
 
# Build cherryd inside the container.
RUN go install github.com/superkkt/cherry/cherryd

# Run the cherryd command by default when the container starts.
ENTRYPOINT ["/go/bin/cherryd"]

# Document that the service listens on port 6633.
EXPOSE 6633
