package main

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

func getid(domain string, link string, date time.Time) string {
	dateFormatted := fmt.Sprintf("%d-%02d-%02d", date.Year(), date.Month(), date.Day())
	return fmt.Sprintf("tag:%v,%v:%v", domain, dateFormatted, link)
}

func getFiles(dir string) chan fileName {
	c := make(chan fileName)
	go func() {
		files, err := ioutil.ReadDir(dir)
		check(err)
		for _, f := range files {
			fname := f.Name()
			ext := filepath.Ext(fname)
			if ext == ".mp3" {
				c <- fileName(path.Join(dir, fname))
			}
		}
		close(c)
	}()
	return c
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

func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%02f", d.Seconds())
}

func copyFile(src fileName, dst string) {
	srcFile, err := os.Open(string(src))
	check(err)
	defer srcFile.Close()

	destFile, err := os.Create(dst)
	check(err)
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	check(err)

	err = destFile.Sync()
	check(err)
}
