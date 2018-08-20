FROM golang:latest as builder
COPY . /go/src/github.com/histrio/rssbook
WORKDIR /go/src/github.com/histrio/rssbook
ARG PROJECT=github.com/histrio/rssbook
ARG RELEASE=0.0.1
RUN mkdir /build
RUN CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -ldflags "-s -w -X ${PROJECT}/pkg/version.Release=${RELEASE} -X ${PROJECT}/pkg/version.Commit=$(shell git rev-parse --short HEAD) -X ${PROJECT}/pkg/version.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')" -o /build/main cmd/rssbookcli/main.go

FROM alpine:latest as downloader
RUN apk add curl tar xz upx
RUN mkdir /build
WORKDIR /build

RUN curl https://johnvansickle.com/ffmpeg/builds/ffmpeg-git-64bit-static.tar.xz -o /tmp/ffmpeg-git-64bit-static.tar.xz
RUN tar -xf /tmp/ffmpeg-git-64bit-static.tar.xz --wildcards --no-anchored 'ffmpeg' --no-anchored 'ffprobe' --strip=1
RUN rm -rf /tmp/ffmpeg-git-64bit-static.tar.xz

RUN curl "https://github.com/upx/upx/releases/download/v3.94/upx-3.94-amd64_linux.tar.xz" -L --fail -o /tmp/upx-3.94-amd64_linux.tar.xz
RUN tar -xf /tmp/upx-3.94-amd64_linux.tar.xz --wildcards --no-anchored 'upx' --strip=1
RUN rm -rf /tmp/upx-3.94-amd64_linux.tar.xz

RUN upx ffmpeg -offmpeg.compressed
RUN upx ffprobe -offprobe.compressed

COPY --from=builder /build/main /build/
RUN upx --best main

FROM scratch
ADD empty /tmp/
COPY --from=downloader /build/ffmpeg.compressed /ffmpeg
COPY --from=downloader /build/ffprobe.compressed /ffprobe
COPY --from=builder /build/main /
ENV PATH /
ENTRYPOINT ["/main", "--src", "/data", "--dst", "/data"]
