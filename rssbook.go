package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	//"strings"
	"time"
)

type Link struct {
	Href   string `xml:"href,attr"`
	Rel    string `xml:"rel,attr"`
	Type   string `xml:"type,attr,omitempty"`
	Title  string `xml:"title,attr,omitempty"`
	Length int    `xml:"length,attr,omitempty"`
}

type Entry struct {
	Title    string    `xml:"title"`
	Id       string    `xml:"id"`
	Updated  time.Time `xml:"updated"`
	LinkList []Link    `xml:"link"`
	Author   Author    `xml:"author"`
	Content  string    `xml:"content"`
}

type Author struct {
	Name  string `xml:"name"`
	Email string `xml:"email"`
}

type Atom1 struct {
	XMLName   xml.Name  `xml:"http://www.w3.org/2005/Atom feed"`
	Title     string    `xml:"title"`
	Author    Author    `xml:"author,omitempty"`
	Id        string    `xml:"id"`
	Subtitle  string    `xml:"subtitle"`
	LinkList  []Link    `xml:"link"`
	Generator string    `xml:"generator"`
	Updated   time.Time `xml:"updated"`
	//Rights   string   `xml:"rights"`
	EntryList []Entry `xml:"entry"`
}

func getid(domain string, link string, date time.Time) string {
	date_formatted := fmt.Sprintf("%d-%02d-%02d", date.Year(), date.Month(), date.Day())
	return fmt.Sprintf("tag:%v,%v:%v", domain, date_formatted, link)
}

func generate() {

	entry := Entry{
		Title:   "Episode1",
		Id:      getid("books.falseprotagonist.me", "/readyplayerone", time.Now()),
		Updated: time.Now(),
		LinkList: []Link{
			Link{Href: "https://falseprotagonist.me", Rel: "alternate"},
			Link{
				Href:   "https://falseprotagonist.me/test.mp3",
				Rel:    "alternate",
				Type:   "audio/mpeg",
				Title:  "MP3",
				Length: 1234,
			},
		},
		Author: Author{
			Name:  "Robert Harrison",
			Email: "rh@rh.rh",
		},
		Content: "test",
	}

	rss := &Atom1{
		Title:    "Ready Player One (Book)",
		Id:       getid("books.falseprotagonist.me", "/readyplayerone", time.Now()),
		Subtitle: "Audiobook as a podcast",
		LinkList: []Link{
			Link{Href: "https://falseprotagonist.me", Rel: "self"},
		},
		Updated:   time.Now(),
		Generator: "rssbook/0.1(+https://github.com/histrio/rssbook)",
		EntryList: []Entry{
			entry,
		},
	}

	out, err := xml.MarshalIndent(rss, "", "  ")

	check(err)

	fmt.Println(string(out))
	//ffmpeg -i out.mp3 -acodec copy -t 00:10:00 -ss 00:05:00 half_hour_split2.mp3
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {

	argsCount := len(os.Args[1:])
	if argsCount != 1 {
		panic("Wrong params")
	}

	dir := os.Args[1]
	files, err := ioutil.ReadDir(dir)

	check(err)

	//filelist := []string{}

	file, err := os.Create("/tmp/dat2")
	check(err)

	for _, f := range files {
		fname := f.Name()
		ext := filepath.Ext(fname)
		if ext == ".mp3" {
			file.WriteString(fmt.Sprintf("file '%v'\n", path.Join(dir, fname)))
			//filelist = append(filelist, "'"+f.Name()+"'")
		}
	}
	file.Close()

	out, err := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", "/tmp/dat2", "-c", "copy", "out.mp3").Output()
	check(err)

	fmt.Println(out)
}
