.PHONY: build docker-build get-ffmpeg

build:
	go build -v -a -installsuffix cgo -o main main.go rss.go utils.go audio.go

docker-build:
	mkdir -p ./build
	docker run --rm -v "${PWD}":/app -w /app -e CGO_ENABLED=0 -e GOOS=linux golang:latest make build
	curl https://johnvansickle.com/ffmpeg/builds/ffmpeg-git-64bit-static.tar.xz -o /tmp/ffmpeg-git-64bit-static.tar.xz
	tar -xf /tmp/ffmpeg-git-64bit-static.tar.xz --wildcards --no-anchored 'ffmpeg' --no-anchored 'ffprobe' --strip=1 
	rm -rf /tmp/ffmpeg-git-64bit-static.tar.xz
	docker build -f Dockerfile.run -t histrio/rssbook:latest -t histrio/rssbook:${VER} .
	docker push histrio/rssbook:${VER}
	docker push histrio/rssbook:latest
