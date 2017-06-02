FROM golang:onbuild
CMD ["CGO_ENABLED=0", "GOOS=linux", "go", "build", "-a", "-installsuffix", "cgo", "-o", "main", main.go, rss.go, utils.go, audio.go]
