.PHONY: build docker-build docker-publish
VOLUME_NAME = my-data
RSSBOOK_S3_BUCKET = s3://files.false.org.ru/
PROJECT = github.com/histrio/rssbook
BUILDTIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT = $(shell git rev-parse --short HEAD)
VER = 0.0.3

OPTS = -X ${PROJECT}/pkg/version.Release=${VER} -X ${PROJECT}/pkg/version.Commit=${COMMIT} -X ${PROJECT}/pkg/version.BuildTime=${BUILDTIME}
OUTPUT = -o ./build/rssbook cmd/rssbookcli/main.go

build:
	go build -v -ldflags "${OPTS}" ${OUTPUT}

build-native:
	CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -ldflags "-s -w ${OPTS}" ${OUTPUT}

test:
	go test ./... -v -cover

upload:
	s3cmd put -r --storage-class=ONEZONE_IA --acl-public ${SOURCE} s3://files.false.org.ru/

docker-build:
	docker build -t histrio/rssbook:latest -t histrio/rssbook:${VER} .

docker-publish:
	docker push histrio/rssbook:${VER}

start:
	make docker-build
	docker rm ${VOLUME_NAME} || true
	docker create -v "${BOOK_SOURCE}":/data --name ${VOLUME_NAME} busybox /bin/true
	docker run -it --rm \
		--volumes-from ${VOLUME_NAME} histrio/rssbook:${VER} \
		--name "${BOOK_ID}" \
		--author "${BOOK_AUTHOR}" \
		--title "${BOOK_TITLE}"
	docker run -it --rm \
		-e AWS_CREDENTIAL_FILE=/root/.aws/credentials
		--volumes-from ${VOLUME_NAME} \
		--volume ~/.aws:/root/.aws
		cgswong/aws:s3cmd put -r -rr -P /data/${BOOK_ID} ${RSSBOOK_S3_BUCKET}
