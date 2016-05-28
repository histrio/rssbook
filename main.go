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
	infoLog    *log.Logger
	warningLog *log.Logger
	errorLog   *log.Logger
)

func initLoggers(
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	infoLog = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	warningLog = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	errorLog = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

type splitterTask struct {
	pos    int
	source string
	dest   string
	skip   time.Time
	limit  time.Time
}

type bookEpisode struct {
	bookID string
	file   string
	pos    int
}

var terminatorTask = splitterTask{}

type rssLink struct {
	Href   string `xml:"href,attr"`
	Rel    string `xml:"rel,attr"`
	Type   string `xml:"type,attr,omitempty"`
	Title  string `xml:"title,attr,omitempty"`
	Length int64  `xml:"length,attr,omitempty"`
}

type rssEntry struct {
	Title    string    `xml:"title"`
	ID       string    `xml:"id"`
	Updated  time.Time `xml:"updated"`
	LinkList []rssLink `xml:"link"`
	Author   rssAuthor `xml:"author"`
	Content  string    `xml:"content"`
}

type rssAuthor struct {
	Name  string `xml:"name"`
	Email string `xml:"email"`
}

type rssBody struct {
	XMLName   xml.Name  `xml:"http://www.w3.org/2005/Atom feed"`
	Title     string    `xml:"title"`
	Author    rssAuthor `xml:"author,omitempty"`
	ID        string    `xml:"id"`
	Subtitle  string    `xml:"subtitle"`
	LinkList  []rssLink `xml:"link"`
	Generator string    `xml:"generator"`
	Updated   time.Time `xml:"updated"`
	//Rights   string   `xml:"rights"`
	EntryList []rssEntry `xml:"entry"`
}

type entrySorter []rssEntry

func (a entrySorter) Len() int           { return len(a) }
func (a entrySorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a entrySorter) Less(i, j int) bool { return a[i].ID < a[j].ID }

const siteURL string = "https://books.falseprotagonist.me/"
const s3Url string = "https://s3-eu-west-1.amazonaws.com"
const s3Bucket string = "falseprotagonist-one"
const bookAuthor string = "Robert Harryson"

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

func generate(bookID string, episodes chan bookEpisode, result chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	entries := []rssEntry{}

	t0 := time.Now()

	for ep := range episodes {
		n := ep.pos
		_, epFilename := filepath.Split(ep.file)
		epName := fmt.Sprintf("Episode%d", n)
		epSize := getFileSize(ep.file)

		content := fmt.Sprintf("Episode %d for %s", n, bookID)
		href := strings.Join([]string{s3Url, s3Bucket, bookID, epFilename}, "/")
		entry := rssEntry{
			Title:   epName,
			ID:      getid("books.falseprotagonist.me", fmt.Sprintf("%s%d", bookID, n), t0),
			Updated: t0.Add(time.Second * time.Duration(n)),
			LinkList: []rssLink{
				rssLink{Href: siteURL + bookID, Rel: "alternate"},
				rssLink{
					Href:   href,
					Rel:    "alternate",
					Type:   "audio/mpeg",
					Title:  epName,
					Length: epSize,
				},
			},
			Author: rssAuthor{
				Name:  bookAuthor,
				Email: "rh@rh.rh",
			},
			Content: content,
		}
		entries = append(entries, entry)
	}

	sort.Sort(entrySorter(entries))

	selfLink := strings.Join([]string{s3Url, s3Bucket, bookID + ".xml"}, "/")
	rss := &rssBody{
		Title:    "Ready Player One (Book)",
		ID:       getid("books.falseprotagonist.me", bookID, t0),
		Subtitle: "Audiobook as a podcast",
		LinkList: []rssLink{
			rssLink{Href: selfLink, Rel: "self"},
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

func cookAudio(dir string, dest string, bookID string) string {

	files, err := ioutil.ReadDir(dir)
	check(err)

	listFile, err := ioutil.TempFile(os.TempDir(), "prefix")
	defer listFile.Close()
	check(err)

	for _, f := range files {
		fname := f.Name()
		ext := filepath.Ext(fname)
		if ext == ".mp3" {
			listFile.WriteString(fmt.Sprintf("file '%v'\n", path.Join(dir, fname)))
		}
	}

	mergedFile, err := ioutil.TempFile(os.TempDir(), "prefix")
	mergedFilename := mergedFile.Name() + ".mp3"

	infoLog.Println("Merging")
	simpleExec("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", listFile.Name(), "-c", "copy", mergedFilename)
	durationRaw := simpleExec("ffprobe", "-i", mergedFilename, "-show_entries", "format=duration", "-v", "quiet", "-of", "csv")
	duration := strings.Split(durationRaw, ",")[1]

	durationS := strings.Split(duration, ".")
	seconds, err := strconv.ParseInt(durationS[0], 10, 64)
	nseconds, err := strconv.ParseInt(durationS[1], 10, 64)

	t0 := time.Time{}
	t0 = t0.Add(time.Second * time.Duration(seconds))
	t0 = t0.Add(time.Nanosecond * time.Duration(nseconds))

	infoLog.Println("Merged to " + formatTime(t0))

	dest = path.Join(dest, bookID)
	os.Mkdir(dest, 0777)

	t1 := time.Time{}
	infoLog.Println("Spliting")

	s1 := t1.Add(time.Minute * 5)

	runtime.GOMAXPROCS(runtime.NumCPU())
	jobs := runtime.NumCPU() * 2

	tasks := make(chan splitterTask, jobs)
	data := make(chan bookEpisode)
	result := make(chan string, 1)

	var wg sync.WaitGroup

	wg.Add(jobs)
	for i := 0; i < jobs; i++ {
		go runner(bookID, tasks, data, &wg)
	}

	var wg2 sync.WaitGroup
	wg2.Add(1)
	go generate(bookID, data, result, &wg2)

	pos := 0
	for t1.Before(t0) {
		pos = pos + 1
		task := splitterTask{
			pos:    pos,
			source: mergedFilename,
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

	os.Remove(listFile.Name())
	os.Remove(mergedFilename)

	xmlOut := <-result
	return xmlOut
}

func split(source string, dest string, skip time.Time, limit time.Time) string {
	fname := fmt.Sprintf("%v%02d%02d%02d.mp3", "part", skip.Hour(), skip.Minute(), skip.Second())
	fpath := path.Join(dest, fname)

	t1s := formatTime(skip)
	s1s := formatTime(limit)
	infoLog.Println("Split " + t1s)

	splitedFile, err := ioutil.TempFile(os.TempDir(), "rssbook-")
	check(err)

	splitedFilename := splitedFile.Name() + ".mp3"
	simpleExec("ffmpeg", "-y", "-i", source, "-acodec", "copy", "-t", s1s, "-ss", t1s, splitedFilename)
	simpleExec("lame", "-V", "9", "--vbr-new", "-mm", "-h", "-q", "0", "-f", splitedFilename, fpath)
	os.Remove(splitedFilename)
	infoLog.Println("Split ended" + t1s)
	return fpath
}

func runner(bookID string, tasks chan splitterTask, data chan bookEpisode, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		task := <-tasks
		if task == terminatorTask {
			break
		}
		filename := split(task.source, task.dest, task.skip, task.limit)
		episode := bookEpisode{
			pos:  task.pos,
			file: filename,
		}

		data <- episode

	}
}

func main() {
	initLoggers(os.Stdout, os.Stdout, os.Stderr)
	var dst string
	var src string
	var bookID string
	flag.StringVar(&dst, "dst", "", "Generated files destination")
	flag.StringVar(&src, "src", "", "Source of audiofiles")
	flag.StringVar(&bookID, "name", "", "Shortname")
	flag.Parse()

	pwd, err := os.Getwd()
	check(err)

	if dst == "" {
		dst = pwd
	}

	output := cookAudio(src, dst, bookID)

	xmlDest := path.Join(dst, bookID+".xml")

	f, err := os.Create(xmlDest)
	f.WriteString(string(output))
}
