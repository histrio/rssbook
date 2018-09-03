package utils

import (
	"os"
	"testing"
	"time"
)

func TestGetFileSize(t *testing.T) {
	filename := "/tmp/dat2"
	f, err := os.Create(filename)
	Check(err)
	d2 := []byte{115, 111, 109, 101, 10}
	n2, err := f.Write(d2)
	Check(err)
	size := GetFileSize(FileName(filename))
	if size != int64(n2) {
		t.Errorf("Size not right")
	}
}

func TestGetID(t *testing.T) {
	type args struct {
		domain string
		link   string
		date   time.Time
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"simple", args{"example.com", "test", time.Time{}}, "tag:example.com,1-01-01:test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetID(tt.args.domain, tt.args.link, tt.args.date); got != tt.want {
				t.Errorf("GetID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMD5Hash(t *testing.T) {
	type args struct {
		text string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"simple", args{"test"}, "098f6bcd4621d373cade4e832627b4f6"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMD5Hash(tt.args.text); got != tt.want {
				t.Errorf("GetMD5Hash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	type args struct {
		t time.Time
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"simple", args{time.Time{}}, "00:00:00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatTime(tt.args.t); got != tt.want {
				t.Errorf("FormatTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	type args struct {
		d time.Duration
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"simple", args{time.Duration(1000)}, "0.000001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatDuration(tt.args.d); got != tt.want {
				t.Errorf("FormatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
