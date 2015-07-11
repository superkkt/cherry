# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

MAINTAINER Kitae Kim

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/superkkt/cherry/
COPY ./cherryd/cherryd.conf /usr/local/etc/
COPY ./docker_entrypoint.sh /usr/local/bin/

ENV DEBIAN_FRONTEND="noninteractive"

RUN sed -i 's/httpredir.debian.org/ftp.daum.net/g' /etc/apt/sources.list \
 && apt-get update \ 
 && apt-get install -y rsyslog mysql-server \
 && sed -i 's/DB_USER/cherry/g' /usr/local/etc/cherryd.conf \
 && sed -i 's/DB_PASSWORD/openflow/g' /usr/local/etc/cherryd.conf \
 && sed -i 's/DB_NAME/cherry/g' /usr/local/etc/cherryd.conf \
 && /etc/init.d/rsyslog start \
 && /etc/init.d/mysql start \
 && echo "CREATE DATABASE cherry" | mysql -u root \
 && echo "GRANT ALL ON cherry.* TO cherry@'127.0.0.1' IDENTIFIED BY 'openflow'" | mysql -u root
 
# Build cherryd inside the container.
RUN go get github.com/superkkt/cherry/cherryd \
 && go install github.com/superkkt/cherry/cherryd

# Run the cherryd command by default when the container starts.
ENTRYPOINT ["/usr/local/bin/docker_entrypoint.sh"]

# Document that the service listens on port 6633.
EXPOSE 6633
