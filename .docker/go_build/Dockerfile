FROM centos:7

RUN yum update -y
RUN yum install -y wget unzip git

ARG GO_VERSION='1.13.5'
ARG GO_BINARY='go'"${GO_VERSION}"'.linux-amd64.tar.gz'
RUN wget https://dl.google.com/go/"${GO_BINARY}" -O /tmp/"${GO_BINARY}"
RUN tar -C /usr/local -xzf /tmp/"${GO_BINARY}"
ENV GOROOT=/usr/local/go
RUN echo 'export GOROOT="'"${GOROOT}"'"' >> /etc/profile.d/myGo.sh
ENV GOPATH=/usr/local/gopath
RUN mkdir -p ${GOPATH}
RUN echo 'export GOPATH="'"${GOPATH}"'"' >> /etc/profile.d/myGo.sh
RUN echo 'export PATH=${PATH}:${GOROOT}/bin:${GOPATH}/bin' >> /etc/profile.d/myGo.sh
ENV PATH=${PATH}:${GOROOT}/bin:${GOPATH}/bin

VOLUME /target_data

ENV TZ=Asia/Tokyo
