# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

MAINTAINER Kitae Kim

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/superkkt/cherry/
COPY ./cherryd/cherryd.conf /usr/local/etc/
COPY ./docker_entrypoint.sh /entrypoint.sh

ENV DEBIAN_FRONTEND="noninteractive"

RUN sed -i 's/httpredir.debian.org/ftp.daum.net/g' /etc/apt/sources.list && \
    apt-get update && \ 
    apt-get install -y rsyslog
 
# Build cherryd inside the container.
RUN go get github.com/superkkt/cherry/cherryd \
 && go install github.com/superkkt/cherry/cherryd

VOLUME /var/log

# Run the cherryd command by default when the container starts.
ENTRYPOINT ["/entrypoint.sh"]

# Document that the service listens on port 6633.
EXPOSE 6633
