package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
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

type Silence struct {
	Start    time.Duration
	End      time.Duration
	Duration time.Duration
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

// GetID generate id from args
func GetID(domain string, link string, date time.Time) string {
	dateFormatted := fmt.Sprintf("%d-%02d-%02d", date.Year(), date.Month(), date.Day())
	return fmt.Sprintf("tag:%v,%v:%v", domain, dateFormatted, link)
}

// GetFiles returns a channel with files in directory. Ordered by names and subfolders included.
func GetFiles(dir string) chan FileName {
	c := make(chan FileName)
	go func() {

		e := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
			ext := filepath.Ext(path)
			if ext == ".mp3" {
				log.Println("Processing: " + path)
				c <- FileName(path)
			}
			return err
		})
		Check(e)
		close(c)
	}()
	return c
}

// GetFileSize calculate file's size
func GetFileSize(fn FileName) int64 {
	file, err := os.Open(string(fn))
	Check(err)
	fi, err := file.Stat()
	Check(err)
	return fi.Size()
}

// Check checks last error
func Check(e error) {
	if e != nil {
		log.Fatalf("%s\n", e)
	}
}

// SimpleExec executes command with args
func SimpleExec(name string, arg ...string) string {
	cmd := exec.Command(name, arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + string(output))
		return "Error"
	}
	return string(output)
}

// FormatTime formats time into 00:00:00
func FormatTime(t time.Time) string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

// FormatDuration formats Duration into seconds
func FormatDuration(d time.Duration) string {
	return fmt.Sprintf("%02f", d.Seconds())
}

// CopyFile copyes a file
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

// GetMD5Hash calculates md5 for a string
func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
