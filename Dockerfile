FROM scratch
ADD empty /tmp/
ADD ./build/ffmpeg.compressed /ffmpeg
ADD ./build/ffprobe.compressed /ffprobe
ADD ./build/main /
ENV PATH /
ENTRYPOINT ["/main", "--src", "/data", "--dst", "/data"]
