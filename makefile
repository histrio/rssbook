.PHONY: build docker-build get-ffmpeg

build:
	go build -v -a -installsuffix cgo -o ./build/main main.go rss.go utils.go audio.go

docker-build:
	mkdir -p ./build
	docker run --rm -v "${PWD}":/app -w /app -e CGO_ENABLED=0 -e GOOS=linux golang:latest make build
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
	docker build --no-cache -t histrio/rssbook:latest -t histrio/rssbook:${VER} .
	docker push histrio/rssbook:${VER}
	docker push histrio/rssbook:latest
