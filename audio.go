package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

// Calculate duration of audio file
func getDuration(filename fileName) time.Duration {
	durationRaw := simpleExec("ffprobe", "-i", string(filename), "-show_entries", "format=duration", "-v", "quiet", "-of", "csv")
	duration := strings.Split(durationRaw, ",")[1]
	durationS := strings.Split(duration, ".")
	seconds, err := strconv.ParseInt(strings.TrimSpace(durationS[0]), 10, 64)
	check(err)
	nseconds, err := strconv.ParseInt(strings.TrimSpace(durationS[1]), 10, 64)
	check(err)
	return time.Second*time.Duration(seconds) + time.Nanosecond*time.Duration(nseconds)
}

func getSplittedEpisodes(in <-chan fileName, limitMin int) chan splitPlan {
	episodeLimit := time.Duration(limitMin) * time.Minute
	plan := make(chan splitPlan)
	pos := 1
	go func() {
		splits := []fileSplit{}
		debt := time.Duration(0)
		for f := range in {
			duration := getDuration(f)
			t0 := time.Duration(0)
			if debt > time.Duration(0) {
				if debt <= duration {
					splits = append(splits, fileSplit{inputFile: f, from: t0, to: t0 + debt})
					plan <- splits
					pos = pos + 1
					splits = []fileSplit{}
					t0 = debt
					debt = time.Duration(0)
				}
				if debt > duration {
					splits = append(splits, fileSplit{
						inputFile: f, from: t0, to: duration})
					debt = debt - duration
					continue
				}
			}
			for (t0 + episodeLimit) < duration {
				splits = append(splits, fileSplit{inputFile: f, from: t0, to: t0 + episodeLimit})
				plan <- splits
				pos = pos + 1
				splits = []fileSplit{}
				t0 = t0 + episodeLimit
			}
			splits = append(splits, fileSplit{inputFile: f, from: t0, to: duration})
			debt = episodeLimit - (duration - t0)
		}
		plan <- splits
		close(plan)
	}()

	return plan
}

func getMergedEpisodes(in <-chan splitPlan) chan fileName {
	c := make(chan fileName)
	go func() {
		for episode := range in {
			listFile, err := ioutil.TempFile(os.TempDir(), "rssbook_mergelist_")
			check(err)
			temp := []string{}
			for _, split := range episode {
				tempFile, err := ioutil.TempFile(os.TempDir(), "rssbook_split_")
				check(err)
				name := tempFile.Name()
				temp = append(temp, name)
				simpleExec("ffmpeg", "-y", "-i", string(split.inputFile), "-acodec", "copy", "-f", "mp3",
					"-ss", formatDuration(split.from),
					"-to", formatDuration(split.to),
					"-write_xing", "0", name)
				listFile.WriteString(fmt.Sprintf("file '%v'\n", name))
			}
			listFile.Close()
			ep, err := ioutil.TempFile(os.TempDir(), "rssbook_concat_")
			check(err)
			simpleExec("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", listFile.Name(), "-f", "mp3", "-c", "copy", ep.Name())

			go func() {
				os.Remove(listFile.Name())
				for _, item := range temp {
					os.Remove(item)
				}
			}()

			c <- fileName(ep.Name())
		}
		close(c)
	}()
	return c
}

func getComressedEpisodes(in <-chan fileName) chan fileName {
	c := make(chan fileName)
	go func() {
		for ep := range in {
			listFile, err := ioutil.TempFile(os.TempDir(), "rssbook_compress_")
			check(err)
			simpleExec("ffmpeg", "-y", "-i", string(ep), "-codec:a", "libmp3lame", "-qscale:a", "9", "-f", "mp3", listFile.Name())
			go os.Remove(string(ep))
			c <- fileName(listFile.Name())
		}
		close(c)
	}()
	return c
}

func getAudioMeta(file fileName) audioMeta {
	metaFile, err := ioutil.TempFile(os.TempDir(), "rssbook_meta_")
	defer metaFile.Close()
	defer os.Remove(metaFile.Name())
	check(err)
	simpleExec("ffmpeg", "-y", "-i", string(file), "-f", "ffmetadata", metaFile.Name())
	f, err := os.Open(metaFile.Name())
	scanner := bufio.NewScanner(f)
	result := audioMeta{}
	for scanner.Scan() {
		line := strings.SplitN(scanner.Text(), "=", 2)
		if len(line) == 2 {
			attr, value := line[0], line[1]
			switch attr {
			case "artist":
				result.author = value
			case "album":
				result.title = value
			}
		}
	}
	return result
}
