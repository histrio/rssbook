package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
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

func generate(episodes []string) string {

	entries := []Entry{}

	for _, ep := range episodes {
		entry := Entry{
			Title:   "Episode1",
			Id:      getid("books.falseprotagonist.me", "/readyplayerone", time.Now()),
			Updated: time.Now(),
			LinkList: []Link{
				Link{Href: "https://falseprotagonist.me", Rel: "alternate"},
				Link{
					Href:   "https://falseprotagonist.me/" + ep,
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
		entries = append(entries, entry)
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
		EntryList: entries,
	}

	out, err := xml.MarshalIndent(rss, "", "  ")

	check(err)

	return string(out)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func simple_exec(name string, arg ...string) string {
	cmd := exec.Command(name, arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + string(output))
		return "Error"
	} else {
		return string(output)
	}
}

func format_time(t time.Time) string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

func cook_audio(dir string) []string {

	Info := log.New(os.Stdout,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	files, err := ioutil.ReadDir(dir)

	check(err)

	list_file, err := ioutil.TempFile(os.TempDir(), "prefix")
	check(err)

	for _, f := range files {
		fname := f.Name()
		ext := filepath.Ext(fname)
		if ext == ".mp3" {
			list_file.WriteString(fmt.Sprintf("file '%v'\n", path.Join(dir, fname)))
		}
	}
	list_file.Close()

	merged_file, err := ioutil.TempFile(os.TempDir(), "prefix")
	merged_filename := merged_file.Name() + ".mp3"

	Info.Println("Merging")
	simple_exec("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", list_file.Name(), "-c", "copy", merged_filename)
	duration_raw := simple_exec("ffprobe", "-i", merged_filename, "-show_entries", "format=duration", "-v", "quiet", "-of", "csv")
	duration := strings.Split(duration_raw, ",")[1]

	Info.Println(duration)

	duration_s := strings.Split(duration, ".")
	seconds, err := strconv.ParseInt(duration_s[0], 10, 64)
	nseconds, err := strconv.ParseInt(duration_s[1], 10, 64)

	t0 := time.Time{}
	t0 = t0.Add(time.Second * time.Duration(seconds))
	t0 = t0.Add(time.Nanosecond * time.Duration(nseconds))

	Info.Println("Merged to " + format_time(t0))

	t1 := time.Time{}
	Info.Println("Spliting")

	data := []string{}
	for t1.Before(t0) {
		s1 := t1
		t1 = t1.Add(time.Minute * 5)

		t1s := format_time(t1)
		s1s := format_time(s1)
		Info.Println("Split " + s1s + " - " + t1s)

		splited_file, err := ioutil.TempFile(os.TempDir(), "prefix")
		check(err)

		splited_filename := splited_file.Name() + ".mp3"
		simple_exec("ffmpeg", "-y", "-i", merged_filename, "-acodec", "copy", "-t", t1s, "-ss", s1s, splited_filename)
		simple_exec("lame", "-V", "9", "--vbr-new", "-mm", "-h", "-q", "0", "-f", splited_filename)
		os.Remove(splited_filename)
		data = append(data, splited_filename+".mp3")
	}

	os.Remove(list_file.Name())
	os.Remove(merged_filename)

	return data
}

func main() {
	argsCount := len(os.Args[1:])
	if argsCount != 1 {
		panic("Wrong params")
	}

	dir := os.Args[1]
	episodes := cook_audio(dir)
	output := generate(episodes)
	fmt.Println(output)
}
