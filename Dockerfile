FROM golang:latest as builder

RUN go get github.com/gosimple/slug
RUN mkdir /build

COPY . /go/src/github.com/histrio/rssbook
WORKDIR /go/src/github.com/histrio/rssbook
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -ldflags "-s -w -X ${PROJECT}/pkg/version.Release=${RELEASE} -X ${PROJECT}/pkg/version.Commit=$(git rev-parse --short HEAD) -X ${PROJECT}/pkg/version.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" -o /build/main cmd/rssbookcli/main.go

# ====================================================

FROM alpine:latest as downloader
RUN apk add curl tar xz upx
RUN mkdir /build
WORKDIR /build

RUN curl https://johnvansickle.com/ffmpeg/builds/ffmpeg-git-64bit-static.tar.xz -o /tmp/ffmpeg-git-64bit-static.tar.xz
RUN tar -xf /tmp/ffmpeg-git-64bit-static.tar.xz --wildcards --no-anchored 'ffmpeg' --no-anchored 'ffprobe' --strip=1
RUN rm -rf /tmp/ffmpeg-git-64bit-static.tar.xz

RUN upx ffmpeg -offmpeg.compressed
RUN upx ffprobe -offprobe.compressed

COPY --from=builder /build/main /build/
RUN upx --best main

# ====================================================

FROM scratch
ADD empty /tmp/
COPY --from=downloader /build/ffmpeg.compressed /ffmpeg
COPY --from=downloader /build/ffprobe.compressed /ffprobe
COPY --from=builder /build/main /
ENV PATH /
ENTRYPOINT ["/main", "--src", "/data", "--dst", "/data"]
