.PHONY: build docker-build get-ffmpeg
VOLUME_NAME = my-data
RSSBOOK_S3_BUCKET = s3://files.falseprotagonist.me/
PROJECT = github.com/histrio/rssbook
BUILDTIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT = $(shell git rev-parse --short HEAD)
RELEASE = 0.0.3

build:
	CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -ldflags "-s -w -X ${PROJECT}/pkg/version.Release=${RELEASE} -X ${PROJECT}/pkg/version.Commit=${COMMIT} -X ${PROJECT}/pkg/version.BuildTime=${BUILDTIME}" -o ./build/rssbook cmd/rssbookcli/main.go

start:
	docker create -v "${BOOK_SOURCE}":/data --name ${VOLUME_NAME} busybox /bin/true
	docker run -it --rm -e BUILDTIME=${BUILDTIME} -e COMMIT=${COMMIT} -e PROJECT=${PROJECT} -e RELEASE=${RELEASE} --privileged --volumes-from ${VOLUME_NAME} histrio/rssbook:latest \
		--name "${BOOK_ID}" --author "${BOOK_AUTHOR}" --title "${BOOK_TITLE}"
	docker run -it --rm -e AWS_CREDENTIAL_FILE=/root/.aws/credentials --volumes-from ${VOLUME_NAME} \
		--volume ~/.aws:/root/.aws cgswong/aws:s3cmd put -r -rr -P /data/${BOOK_ID} ${RSSBOOK_S3_BUCKET}
	docker rm ${VOLUME_NAME}

docker-build:
	docker build -t histrio/rssbook:latest -t histrio/rssbook:${VER} .
	docker push histrio/rssbook:${VER}
	docker push histrio/rssbook:latest
