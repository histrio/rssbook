FROM golang:latest as builder

RUN go get github.com/gosimple/slug
RUN mkdir /build

COPY . /go/src/github.com/histrio/rssbook
WORKDIR /go/src/github.com/histrio/rssbook
RUN make test
RUN make build

# ====================================================

FROM alpine:latest as downloader
RUN apk add curl tar xz upx

RUN curl https://johnvansickle.com/ffmpeg/builds/ffmpeg-git-amd64-static.tar.xz -o /tmp/ffmpeg-git-64bit-static.tar.xz
RUN tar -xf /tmp/ffmpeg-git-64bit-static.tar.xz --wildcards --no-anchored 'ffmpeg' --no-anchored 'ffprobe' --strip=1
RUN rm -rf /tmp/ffmpeg-git-64bit-static.tar.xz

RUN upx ffmpeg -o /ffmpeg.compressed
RUN upx ffprobe -o /ffprobe.compressed

COPY --from=builder /go/src/github.com/histrio/rssbook/build/rssbook /
RUN upx --best /rssbook

# ====================================================

FROM scratch
ADD empty /tmp/
COPY --from=downloader /ffmpeg.compressed /ffmpeg
COPY --from=downloader /ffprobe.compressed /ffprobe
COPY --from=downloader /rssbook /
ENV PATH /
ENTRYPOINT ["/rssbook", "--src", "/data", "--dst", "/data"]
