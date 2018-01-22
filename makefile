.PHONY: build docker-build get-ffmpeg
VOLUME_NAME = my-data
RSSBOOK_S3_BUCKET = s3://files.falseprotagonist.me/

RELEASE?=0.0.1
COMMIT?=$(shell git rev-parse --short HEAD)
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
PROJECT?=github.com/histrio/rssbook

build:
	go build -v -a -installsuffix cgo -ldflags "-s -w -X ${PROJECT}/pkg/version.Release=${RELEASE} -X ${PROJECT}/pkg/version.Commit=${COMMIT} -X ${PROJECT}/pkg/version.BuildTime=${BUILD_TIME}" -o ./build/main cmd/rssbookcli/main.go

start:
	docker create -v "${BOOK_SOURCE}":/data --name ${VOLUME_NAME} busybox /bin/true
	docker run -it --rm --privileged --volumes-from ${VOLUME_NAME} histrio/rssbook:latest \
		--name "${BOOK_ID}" --author "${BOOK_AUTHOR}" --title "${BOOK_TITLE}"
	docker run -it --rm -e AWS_CREDENTIAL_FILE=/root/.aws/credentials --volumes-from ${VOLUME_NAME} --volume ~/.aws:/root/.aws cgswong/aws:s3cmd put -r -rr -P /data/${BOOK_ID} ${RSSBOOK_S3_BUCKET}
	docker rm ${VOLUME_NAME}

docker-build:
	mkdir -p ./build
	docker run --rm -v "${PWD}":/go/src/github.com/histrio/rssbook -w /go/src/github.com/histrio/rssbook -e CGO_ENABLED=0 -e GOOS=linux golang:latest make build
ifeq ("$(wildcard ./build/ffmpeg)","")
	curl https://johnvansickle.com/ffmpeg/builds/ffmpeg-git-64bit-static.tar.xz -o /tmp/ffmpeg-git-64bit-static.tar.xz
	cd ./build && tar -xf /tmp/ffmpeg-git-64bit-static.tar.xz --wildcards --no-anchored 'ffmpeg' --no-anchored 'ffprobe' --strip=1
	rm -rf /tmp/ffmpeg-git-64bit-static.tar.xz
endif
ifeq ("$(wildcard ./build/upx)","")
	curl "https://github.com/upx/upx/releases/download/v3.94/upx-3.94-amd64_linux.tar.xz" -L --fail -o /tmp/upx-3.94-amd64_linux.tar.xz
	cd ./build && tar -xf /tmp/upx-3.94-amd64_linux.tar.xz --wildcards --no-anchored 'upx' --strip=1
	rm -rf /tmp/upx-3.94-amd64_linux.tar.xz
endif
ifeq ("$(wildcard ./build/ffprobe.compressed)","")
	cd ./build && ./upx --best ffprobe -offprobe.compressed
endif
ifeq ("$(wildcard ./build/ffmpeg.compressed)","")
	cd ./build && ./upx --best ffmpeg -offmpeg.compressed
endif
	cd ./build && ./upx --best main
	docker build -t histrio/rssbook:latest -t histrio/rssbook:${VER} .
	docker push histrio/rssbook:${VER}
	docker push histrio/rssbook:latest
