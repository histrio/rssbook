package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// Merge all audio files listed in file into one
func mergeFiles(listFileName string) string {
	infoLog.Println("Merging")
	mergedFile, err := ioutil.TempFile(os.TempDir(), "prefix")
	check(err)
	mergedFilename := mergedFile.Name() + ".mp3"
	simpleExec("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", listFileName, "-c", "copy", mergedFilename)
	return mergedFilename
}

// Calculate duration of audio file
func getDuration(filename string) time.Time {
	durationRaw := simpleExec("ffprobe", "-i", filename, "-show_entries", "format=duration", "-v", "quiet", "-of", "csv")
	duration := strings.Split(durationRaw, ",")[1]
	durationS := strings.Split(duration, ".")
	seconds, err := strconv.ParseInt(strings.TrimSpace(durationS[0]), 10, 64)
	check(err)
	nseconds, err := strconv.ParseInt(strings.TrimSpace(durationS[1]), 10, 64)
	check(err)
	t0 := time.Time{}
	t0 = t0.Add(time.Second * time.Duration(seconds))
	t0 = t0.Add(time.Nanosecond * time.Duration(nseconds))
	infoLog.Println("Duration: " + formatTime(t0))
	return t0
}

func split(book bookMeta, skip time.Time, limit time.Time, pos int) string {
	name := fmt.Sprintf("part%02d%02d%02d", skip.Hour(), skip.Minute(), skip.Second())
	fname := fmt.Sprintf("%s.mp3", name)
	fpath := path.Join(book.dst, fname)

	t1s := formatTime(skip)
	s1s := formatTime(limit)
	infoLog.Println("Split " + t1s)

	splitedFile, err := ioutil.TempFile(os.TempDir(), "rssbook-")
	check(err)

	splitedFilename := splitedFile.Name() + ".mp3"
	defer os.Remove(splitedFilename)

	albumMetadata := fmt.Sprintf("album=%v", book.title)
	authorMetadata := fmt.Sprintf("artist=%v", book.author)
	titleMetadata := fmt.Sprintf("title=%v", name)
	trackMetadata := fmt.Sprintf("track=%d", pos)

	simpleExec("ffmpeg", "-y", "-i", book.src, "-write_xing", "0", "-metadata", trackMetadata, "-metadata", albumMetadata, "-metadata", titleMetadata, "-metadata", authorMetadata, "-acodec", "copy", "-t", s1s, "-ss", t1s, splitedFilename)
	simpleExec("lame", "-V", "9", "--vbr-new", "-mm", "-h", "-q", "0", "-f", splitedFilename, fpath)
	return fpath
}
