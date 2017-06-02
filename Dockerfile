FROM scratch
ADD empty /tmp/
ADD ./build/ffmpeg /
ADD ./build/ffprobe /
ADD ./build/main /
ENV PATH /
ENTRYPOINT ["/main"]
