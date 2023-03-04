FROM golang:1.20 AS builder

RUN mkdir /.cache && chmod 777 /.cache
RUN apt update && apt install zip -y
WORKDIR /
ENV TZ=Asia/Tokyo
