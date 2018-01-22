package utils

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"
)

const SiteURL string = "https://www.falseprotagonist.me/"
const S3Url string = "http://files.falseprotagonist.me/"
const S3Bucket string = "falseprotagonist-one/"

type FileSplit struct {
	InputFile FileName
	From      time.Duration
	To        time.Duration
}

type AudioMeta struct {
	Author string
	Title  string
}

type SplitPlan []FileSplit
type FileName string

type BookMeta struct {
	Id       string
	Title    string
	Author   string
	Episodes episodesList
}

type BookEpisode struct {
	Name     string
	Href     string
	N        string
	Pos      int
	File     string
	FileSize int64
	Duration time.Duration
}

type episodesList []BookEpisode

func Getid(domain string, link string, date time.Time) string {
	dateFormatted := fmt.Sprintf("%d-%02d-%02d", date.Year(), date.Month(), date.Day())
	return fmt.Sprintf("tag:%v,%v:%v", domain, dateFormatted, link)
}

func GetFiles(dir string) chan FileName {
	c := make(chan FileName)
	go func() {
		files, err := ioutil.ReadDir(dir)
		Check(err)
		for _, f := range files {
			fname := f.Name()
			ext := filepath.Ext(fname)
			if ext == ".mp3" {
				c <- FileName(path.Join(dir, fname))
			}
		}
		close(c)
	}()
	return c
}

func GetFileSize(fn FileName) int64 {
	file, err := os.Open(string(fn))
	Check(err)
	fi, err := file.Stat()
	Check(err)
	return fi.Size()
}

func Check(e error) {
	if e != nil {
		log.Fatalf("%s\n", e)
	}
}

func SimpleExec(name string, arg ...string) string {
	cmd := exec.Command(name, arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + string(output))
		return "Error"
	}
	return string(output)
}

func FormatTime(t time.Time) string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

func FormatDuration(d time.Duration) string {
	return fmt.Sprintf("%02f", d.Seconds())
}

func CopyFile(src FileName, dst string) {
	srcFile, err := os.Open(string(src))
	Check(err)
	defer srcFile.Close()

	destFile, err := os.Create(dst)
	Check(err)
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	Check(err)

	err = destFile.Sync()
	Check(err)
}
