package main

import (
	"os"
	"testing"
	"time"
)

func TestGetID(t *testing.T) {
	actual := getid("example.com", "test", time.Time{})
	if actual != "tag:example.com,1-01-01:test" {
		t.Errorf("Tag not right")
	}
}

func TestGetFileSize(t *testing.T) {
	fileName := "/tmp/dat2"
	f, err := os.Create(fileName)
	check(err)
	d2 := []byte{115, 111, 109, 101, 10}
	n2, err := f.Write(d2)
	check(err)
	size := getFileSize(fileName)
	if size != int64(n2) {
		t.Errorf("Size not right")
	}
}
