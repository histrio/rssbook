package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"
)

func getid(domain string, link string, date time.Time) string {
	dateFormatted := fmt.Sprintf("%d-%02d-%02d", date.Year(), date.Month(), date.Day())
	return fmt.Sprintf("tag:%v,%v:%v", domain, dateFormatted, link)
}

func getFileSize(fn string) int64 {
	file, err := os.Open(fn)
	check(err)
	fi, err := file.Stat()
	check(err)
	return fi.Size()
}

func check(e error) {
	if e != nil {
		log.Fatalf("%s\n", e)
	}
}

func simpleExec(name string, arg ...string) string {
	cmd := exec.Command(name, arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + string(output))
		return "Error"
	}
	return string(output)
}

func formatTime(t time.Time) string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

// Collects files from folder to merge
func getFileList(dir string) []string {
	files, err := ioutil.ReadDir(dir)
	check(err)

	var result []string

	for _, f := range files {
		fname := f.Name()
		ext := filepath.Ext(fname)
		if ext == ".mp3" {
			result = append(result, path.Join(dir, fname))
		}
	}
	return result
}

func getFileListFile(dir string) string {
	listFile, err := ioutil.TempFile(os.TempDir(), "prefix")
	defer listFile.Close()
	check(err)

	for _, f := range getFileList(dir) {
		listFile.WriteString(fmt.Sprintf("file '%v'\n", f))
	}
	return listFile.Name()
}
