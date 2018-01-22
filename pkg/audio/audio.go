package audio

import (
	"bufio"
	"fmt"
	"github.com/histrio/rssbook/pkg/utils"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

// Calculate duration of audio file
func GetDuration(filename utils.FileName) time.Duration {
	durationRaw := utils.SimpleExec("ffprobe", "-i", string(filename), "-show_entries", "format=duration", "-v", "quiet", "-of", "csv")
	duration := strings.Split(durationRaw, ",")[1]
	durationS := strings.Split(duration, ".")
	seconds, err := strconv.ParseInt(strings.TrimSpace(durationS[0]), 10, 64)
	utils.Check(err)
	nseconds, err := strconv.ParseInt(strings.TrimSpace(durationS[1]), 10, 64)
	utils.Check(err)
	return time.Second*time.Duration(seconds) + time.Nanosecond*time.Duration(nseconds)
}

func GetSplittedEpisodes(in <-chan utils.FileName, limitMin int) chan utils.SplitPlan {
	episodeLimit := time.Duration(limitMin) * time.Minute
	plan := make(chan utils.SplitPlan)
	pos := 1
	go func() {
		splits := []utils.FileSplit{}
		debt := time.Duration(0)
		for f := range in {
			duration := GetDuration(f)
			t0 := time.Duration(0)
			if debt > time.Duration(0) {
				if debt <= duration {
					splits = append(splits, utils.FileSplit{InputFile: f, From: t0, To: t0 + debt})
					plan <- splits
					pos = pos + 1
					splits = []utils.FileSplit{}
					t0 = debt
					debt = time.Duration(0)
				}
				if debt > duration {
					splits = append(splits, utils.FileSplit{
						InputFile: f, From: t0, To: duration})
					debt = debt - duration
					continue
				}
			}
			for (t0 + episodeLimit) < duration {
				splits = append(splits, utils.FileSplit{InputFile: f, From: t0, To: t0 + episodeLimit})
				plan <- splits
				pos = pos + 1
				splits = []utils.FileSplit{}
				t0 = t0 + episodeLimit
			}
			splits = append(splits, utils.FileSplit{InputFile: f, From: t0, To: duration})
			debt = episodeLimit - (duration - t0)
		}
		plan <- splits
		close(plan)
	}()

	return plan
}

func GetMergedEpisodes(in <-chan utils.SplitPlan) chan utils.FileName {
	c := make(chan utils.FileName)
	go func() {
		for episode := range in {
			listFile, err := ioutil.TempFile(os.TempDir(), "rssbook_mergelist_")
			utils.Check(err)
			temp := []string{}
			for _, split := range episode {
				tempFile, err := ioutil.TempFile(os.TempDir(), "rssbook_split_")
				utils.Check(err)
				name := tempFile.Name()
				temp = append(temp, name)
				utils.SimpleExec("ffmpeg", "-y", "-i", string(split.InputFile), "-acodec", "copy", "-f", "mp3",
					"-ss", utils.FormatDuration(split.From),
					"-to", utils.FormatDuration(split.To),
					"-write_xing", "0", name)
				listFile.WriteString(fmt.Sprintf("file '%v'\n", name))
			}
			listFile.Close()
			ep, err := ioutil.TempFile(os.TempDir(), "rssbook_concat_")
			utils.Check(err)
			utils.SimpleExec("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", listFile.Name(), "-f", "mp3", "-c", "copy", ep.Name())

			go func() {
				os.Remove(listFile.Name())
				for _, item := range temp {
					os.Remove(item)
				}
			}()

			c <- utils.FileName(ep.Name())
		}
		close(c)
	}()
	return c
}

func GetCompressedEpisodes(in <-chan utils.FileName) chan utils.FileName {
	c := make(chan utils.FileName)
	go func() {
		for ep := range in {
			listFile, err := ioutil.TempFile(os.TempDir(), "rssbook_compress_")
			utils.Check(err)
			utils.SimpleExec("ffmpeg", "-y", "-i", string(ep), "-codec:a", "libmp3lame", "-qscale:a", "9", "-f", "mp3", listFile.Name())
			go os.Remove(string(ep))
			c <- utils.FileName(listFile.Name())
		}
		close(c)
	}()
	return c
}

func getAudioMeta(file utils.FileName) utils.AudioMeta {
	metaFile, err := ioutil.TempFile(os.TempDir(), "rssbook_meta_")
	defer metaFile.Close()
	defer os.Remove(metaFile.Name())
	utils.Check(err)
	utils.SimpleExec("ffmpeg", "-y", "-i", string(file), "-f", "ffmetadata", metaFile.Name())
	f, err := os.Open(metaFile.Name())
	scanner := bufio.NewScanner(f)
	result := utils.AudioMeta{}
	for scanner.Scan() {
		line := strings.SplitN(scanner.Text(), "=", 2)
		if len(line) == 2 {
			attr, value := line[0], line[1]
			switch attr {
			case "artist":
				result.Author = value
			case "album":
				result.Title = value
			}
		}
	}
	return result
}
