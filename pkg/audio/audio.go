package audio

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/histrio/rssbook/pkg/utils"
)

// GetDuration Calculate duration of audio file
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

func strToDuration(s string) time.Duration {
	durationS := strings.Split(s, ".")
	var nseconds int64
	seconds, err := strconv.ParseInt(strings.TrimSpace(durationS[0]), 10, 64)
	utils.Check(err)
	if len(durationS) > 1 {
		n, err := strconv.ParseInt(strings.TrimSpace(durationS[1]), 10, 64)
		utils.Check(err)
		nseconds = n
	}
	return time.Second*time.Duration(seconds) + time.Nanosecond*time.Duration(nseconds)
}

// GetSilences returns silences in file
func GetSilences(filename utils.FileName) []utils.Silence {
	rStart := regexp.MustCompile(`silence_start: (\d+(\.\d+)?)`)
	rEndDuration := regexp.MustCompile(`silence_end: (\d+(\.\d+)?) \| silence_duration: (\d+(\.\d+)?)`)

	var result []utils.Silence
	res := utils.SimpleExec("ffmpeg", "-i", string(filename), "-af", "silencedetect=noise=-30dB:d=0.7", "-f", "null", "-")
	var silence utils.Silence
	silence = utils.Silence{}
	for _, s := range strings.Split(res, "\n") {
		if strings.HasPrefix(s, "[silence") {
			s2 := s[33:]
			if strings.HasPrefix(s2, "silence_start") {
				strStart := rStart.FindStringSubmatch(s2)[1]
				silence.Start = strToDuration(strStart)
			} else if strings.HasPrefix(s2, "silence_end") {
				sub := rEndDuration.FindStringSubmatch(s2)
				silence.End = strToDuration(sub[1])
				silence.Duration = strToDuration(sub[3])
				result = append(result, silence)
				silence = utils.Silence{}
			}
		}
	}
	return result
}

func alignSilence(silences []utils.Silence, t time.Duration) time.Duration {

	type Distance struct {
		t time.Duration
		d float64
	}
	var distances []Distance

	for _, a := range silences {
		distance := math.Abs(float64((a.Start - t).Nanoseconds()))
		distances = append(distances, Distance{t: a.Start + (a.Duration), d: distance})
	}
	sort.Slice(distances, func(i, j int) bool {
		return distances[i].d < distances[j].d
	})
	log.Println(t)
	log.Println(distances[0].t)
	log.Println("---")
	return distances[0].t
}

// GetSplittedEpisodes returns split plan
func GetSplittedEpisodes(in <-chan utils.FileName, limitMin int) chan utils.SplitPlan {
	episodeLimit := time.Duration(limitMin) * time.Minute
	plan := make(chan utils.SplitPlan)
	go func() {
		splits := []utils.FileSplit{}
		debt := time.Duration(0)
		for f := range in {
			silences := GetSilences(f)
			duration := GetDuration(f)
			t0 := time.Duration(0)

			// If debt exists
			if debt > time.Duration(0) {
				// And if debt less then duration we will make a split
				// fill the debt and start a new split
				if debt <= duration {
					splits = append(splits, utils.FileSplit{
						InputFile: f,
						From:      t0,
						To:        alignSilence(silences, t0+debt)})
					plan <- splits
					splits = []utils.FileSplit{}
					t0 = alignSilence(silences, debt)
					debt = time.Duration(0)
				}
				// And if debt more then file duration we will take all file and decrease
				// a debt
				if debt > duration {
					splits = append(splits, utils.FileSplit{
						InputFile: f,
						From:      t0,
						To:        duration})
					debt = debt - duration
					continue
				}
			}
			// If episode length is fits in current file
			for (t0 + episodeLimit) < duration {
				splits = append(splits, utils.FileSplit{
					InputFile: f,
					From:      t0,
					To:        alignSilence(silences, t0+episodeLimit)})
				plan <- splits
				splits = []utils.FileSplit{}
				t0 = alignSilence(silences, t0+episodeLimit)
			}
			// Take all the rest as a split
			splits = append(splits, utils.FileSplit{
				InputFile: f,
				From:      t0,
				To:        duration})
			debt = episodeLimit - (duration - t0)
		}
		plan <- splits
		close(plan)
	}()

	return plan
}

// GetMergedEpisodes merge and return by split plan
func GetMergedEpisodes(in <-chan utils.SplitPlan) chan utils.FileName {
	c := make(chan utils.FileName)
	go func() {
		for episode := range in {
			log.Println(episode)
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

// GetCompressedEpisodes compress audio files
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
