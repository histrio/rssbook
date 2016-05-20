package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

func Init(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	Trace = log.New(traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

type Task struct {
	pos    int
	source string
	dest   string
	skip   time.Time
	limit  time.Time
}

type Episode struct {
	file string
	pos  int
}

var terminatorTask = Task{}

type Link struct {
	Href   string `xml:"href,attr"`
	Rel    string `xml:"rel,attr"`
	Type   string `xml:"type,attr,omitempty"`
	Title  string `xml:"title,attr,omitempty"`
	Length int64  `xml:"length,attr,omitempty"`
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

type EntrySorter []Entry

func (a EntrySorter) Len() int           { return len(a) }
func (a EntrySorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EntrySorter) Less(i, j int) bool { return a[i].Id < a[j].Id }

const siteUrl string = "https://books.falseprotagonist.me/"
const s3Url string = "https://s3-eu-west-1.amazonaws.com"
const s3Bucket string = "falseprotagonist-one"
const bookId string = "readyplayerone"
const bookAuthor string = "Robert Harryson"

func getid(domain string, link string, date time.Time) string {
	date_formatted := fmt.Sprintf("%d-%02d-%02d", date.Year(), date.Month(), date.Day())
	return fmt.Sprintf("tag:%v,%v:%v", domain, date_formatted, link)
}

func getFileSize(fn string) int64 {
	file, err := os.Open(fn)
	check(err)
	fi, err := file.Stat()
	check(err)
	return fi.Size()
}

func generate(episodes chan Episode, result chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	entries := []Entry{}

	t0 := time.Now()

	for ep := range episodes {
		n := ep.pos
		_, ep_filename := filepath.Split(ep.file)
		ep_name := fmt.Sprintf("Episode%d", n)
		ep_size := getFileSize(ep.file)

		content := fmt.Sprintf("Episode %d for %s", n, bookId)
		href := strings.Join([]string{s3Url, s3Bucket, bookId, ep_filename}, "/")
		entry := Entry{
			Title:   ep_name,
			Id:      getid("books.falseprotagonist.me", fmt.Sprintf("%s%d", bookId, n), t0),
			Updated: t0.Add(time.Second * time.Duration(n)),
			LinkList: []Link{
				Link{Href: siteUrl + bookId, Rel: "alternate"},
				Link{
					Href:   href,
					Rel:    "alternate",
					Type:   "audio/mpeg",
					Title:  ep_name,
					Length: ep_size,
				},
			},
			Author: Author{
				Name:  bookAuthor,
				Email: "rh@rh.rh",
			},
			Content: content,
		}
		entries = append(entries, entry)
	}

	sort.Sort(EntrySorter(entries))

	selfLink := strings.Join([]string{s3Url, s3Bucket, bookId + ".xml"}, "/")
	rss := &Atom1{
		Title:    "Ready Player One (Book)",
		Id:       getid("books.falseprotagonist.me", bookId, t0),
		Subtitle: "Audiobook as a podcast",
		LinkList: []Link{
			Link{Href: selfLink, Rel: "self"},
		},
		Updated:   t0,
		Generator: "rssbook/0.1(+https://github.com/histrio/rssbook)",
		EntryList: entries,
	}

	out, err := xml.MarshalIndent(rss, "", "  ")

	check(err)

	result <- string(out)
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

func cook_audio(dir string, dest string) string {

	files, err := ioutil.ReadDir(dir)
	check(err)

	list_file, err := ioutil.TempFile(os.TempDir(), "prefix")
	defer list_file.Close()
	check(err)

	for _, f := range files {
		fname := f.Name()
		ext := filepath.Ext(fname)
		if ext == ".mp3" {
			list_file.WriteString(fmt.Sprintf("file '%v'\n", path.Join(dir, fname)))
		}
	}

	merged_file, err := ioutil.TempFile(os.TempDir(), "prefix")
	merged_filename := merged_file.Name() + ".mp3"

	Info.Println("Merging")
	simple_exec("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", list_file.Name(), "-c", "copy", merged_filename)
	duration_raw := simple_exec("ffprobe", "-i", merged_filename, "-show_entries", "format=duration", "-v", "quiet", "-of", "csv")
	duration := strings.Split(duration_raw, ",")[1]

	duration_s := strings.Split(duration, ".")
	seconds, err := strconv.ParseInt(duration_s[0], 10, 64)
	nseconds, err := strconv.ParseInt(duration_s[1], 10, 64)

	t0 := time.Time{}
	t0 = t0.Add(time.Second * time.Duration(seconds))
	t0 = t0.Add(time.Nanosecond * time.Duration(nseconds))

	Info.Println("Merged to " + format_time(t0))

	dest = path.Join(dest, bookId)
	os.Mkdir(dest, 0777)

	t1 := time.Time{}
	Info.Println("Spliting")

	s1 := t1.Add(time.Minute * 5)

	runtime.GOMAXPROCS(runtime.NumCPU())
	jobs := runtime.NumCPU() * 2

	tasks := make(chan Task, jobs)
	data := make(chan Episode)
	result := make(chan string, 1)

	var wg sync.WaitGroup

	wg.Add(jobs)
	for i := 0; i < jobs; i++ {
		go runner(tasks, data, &wg)
	}

	var wg2 sync.WaitGroup
	wg2.Add(1)
	go generate(data, result, &wg2)

	pos := 0
	for t1.Before(t0) {
		pos = pos + 1
		task := Task{
			pos:    pos,
			source: merged_filename,
			dest:   dest,
			skip:   t1,
			limit:  s1,
		}
		tasks <- task
		t1 = t1.Add(time.Minute * 5)
	}

	for i := 0; i < jobs; i++ {
		tasks <- terminatorTask
	}
	close(tasks)
	wg.Wait()

	close(data)
	wg2.Wait()

	os.Remove(list_file.Name())
	os.Remove(merged_filename)

	xml_out := <-result
	return xml_out
}

func split(source string, dest string, skip time.Time, limit time.Time) string {
	fname := fmt.Sprintf("%v%02d%02d%02d.mp3", bookId, skip.Hour(), skip.Minute(), skip.Second())
	fpath := path.Join(dest, fname)

	t1s := format_time(skip)
	s1s := format_time(limit)
	Info.Println("Split " + t1s)

	splited_file, err := ioutil.TempFile(os.TempDir(), "rssbook-"+bookId)
	check(err)

	splited_filename := splited_file.Name() + ".mp3"
	simple_exec("ffmpeg", "-y", "-i", source, "-acodec", "copy", "-t", s1s, "-ss", t1s, splited_filename)
	simple_exec("lame", "-V", "9", "--vbr-new", "-mm", "-h", "-q", "0", "-f", splited_filename, fpath)
	os.Remove(splited_filename)
	Info.Println("Split ended" + t1s)
	return fpath
}

func runner(tasks chan Task, data chan Episode, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		task := <-tasks
		if task == terminatorTask {
			break
		}
		filename := split(task.source, task.dest, task.skip, task.limit)
		episode := Episode{
			pos:  task.pos,
			file: filename,
		}

		data <- episode

	}
}

func main() {
	Init(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	var dst string
	var src string
	var name string
	flag.StringVar(&dst, "dst", "", "Generated files destination")
	flag.StringVar(&src, "src", "", "Source of audiofiles")
	flag.StringVar(&name, "name", "", "Shortname")
	flag.Parse()

	pwd, err := os.Getwd()
	check(err)

	if dst == "" {
		dst = pwd
	}

	output := cook_audio(src, dst)

	xml_dest := path.Join(dst, bookId+".xml")

	f, err := os.Create(xml_dest)
	f.WriteString(string(output))
}
