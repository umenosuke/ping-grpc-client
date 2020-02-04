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

ARG PROTOC_VERSION="3.11.2"
ARG PROTOC_BINARY='protoc-'"${PROTOC_VERSION}"'-linux-x86_64.zip'
RUN wget https://github.com/protocolbuffers/protobuf/releases/download/v"${PROTOC_VERSION}"/"${PROTOC_BINARY}" -O /tmp/"${PROTOC_BINARY}"
RUN unzip -d /usr/local/protoc /tmp/"${PROTOC_BINARY}"
RUN chmod -R +x /usr/local/protoc
RUN echo 'export PATH=${PATH}:/usr/local/protoc/bin' >> /etc/profile.d/myProtoc.sh
ENV PATH=${PATH}:/usr/local/protoc/bin

RUN go get -u github.com/golang/protobuf/protoc-gen-go
RUN go get -u google.golang.org/grpc

RUN go get -u github.com/tmc/grpcutil/protoc-gen-tstypes

VOLUME /target_data

ENV TZ=Asia/Tokyo
