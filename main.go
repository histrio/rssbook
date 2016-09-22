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

type bookMeta struct {
	id  string
	src string
	dst string
}

type bookEpisode struct {
	book bookMeta
	file string
	pos  int
}

type episodesList []bookEpisode

func (slice episodesList) Len() int {
	return len(slice)
}

func (slice episodesList) Less(i, j int) bool {
	return slice[i].pos < slice[j].pos
}

func (slice episodesList) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
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

func generateXML(book bookMeta, episodes episodesList) string {
	infoLog.Println("Generating xml")
	entries := []rssEntry{}
	t0 := time.Now()
	for _, ep := range episodes {
		n := ep.pos
		_, epFilename := filepath.Split(ep.file)
		epName := fmt.Sprintf("Episode%d", n)
		epSize := getFileSize(ep.file)

		content := fmt.Sprintf("Episode %d for %s", n, book.id)
		href := strings.Join([]string{s3Url, s3Bucket, book.id, epFilename}, "/")
		entry := rssEntry{
			Title:   epName,
			ID:      getid("books.falseprotagonist.me", fmt.Sprintf("%s%d", book.id, n), t0),
			Updated: t0.Add(time.Second * time.Duration(n)),
			LinkList: []rssLink{
				rssLink{Href: siteURL + book.id, Rel: "alternate"},
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

	selfLink := strings.Join([]string{s3Url, s3Bucket, book.id + ".xml"}, "/")
	rss := &rssBody{
		Title:    "Ready Player One (Book)",
		ID:       getid("books.falseprotagonist.me", book.id, t0),
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

	xmlDest := path.Join(book.dst, book.id+".xml")
	f, err := os.Create(xmlDest)
	check(err)
	f.WriteString(string(out))
	return xmlDest
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
func getFileList(dir string) string {
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
	return listFile.Name()
}

// Merge all audio files listed in file into one
func mergeFiles(listFileName string) string {
	infoLog.Println("Merging")
	mergedFile, err := ioutil.TempFile(os.TempDir(), "prefix")
	check(err)
	mergedFilename := mergedFile.Name() + ".mp3"
	simpleExec("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", listFileName, "-c", "copy", mergedFilename)
	return mergedFilename
}

// Calculate duration of audio file
func getDuration(filename string) time.Time {
	durationRaw := simpleExec("ffprobe", "-i", filename, "-show_entries", "format=duration", "-v", "quiet", "-of", "csv")
	duration := strings.Split(durationRaw, ",")[1]
	durationS := strings.Split(duration, ".")
	seconds, err := strconv.ParseInt(strings.TrimSpace(durationS[0]), 10, 64)
	check(err)
	nseconds, err := strconv.ParseInt(strings.TrimSpace(durationS[1]), 10, 64)
	check(err)
	t0 := time.Time{}
	t0 = t0.Add(time.Second * time.Duration(seconds))
	t0 = t0.Add(time.Nanosecond * time.Duration(nseconds))
	infoLog.Println("Duration: " + formatTime(t0))
	return t0
}

func splitAsync(book bookMeta, t0 time.Time, src string) episodesList {
	infoLog.Println("Spliting")

	var result episodesList

	t1 := time.Time{}
	s1 := t1.Add(time.Minute * 5)

	runtime.GOMAXPROCS(runtime.NumCPU())
	jobs := runtime.NumCPU() * 2

	tasks := make(chan splitterTask, jobs)
	data := make(chan bookEpisode, 1)

	var wg sync.WaitGroup

	wg.Add(jobs)
	for i := 0; i < jobs; i++ {
		go runner(book.id, tasks, data, &wg)
	}

	go func() {
		defer wg.Done()
		for t := range data {
			result = append(result, t)
		}
	}()

	pos := 0
	for t1.Before(t0) {
		pos = pos + 1
		task := splitterTask{
			pos:    pos,
			source: src,
			dest:   book.dst,
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
	return result
}

func generateM3U(book bookMeta, episodes episodesList) string {
	m3uDest := path.Join(book.dst, book.id+".m3u")
	f, err := os.Create(m3uDest)
	check(err)
	f.WriteString("#EXTM3U\n\n")
	for _, ep := range episodes {
		path, err := filepath.Rel(book.dst, ep.file)
		check(err)
		f.WriteString(path + "\n")
	}
	return m3uDest
}

func cookAudio(book bookMeta) error {
	listFileName := getFileList(book.src)
	defer os.Remove(listFileName)
	mergedFileName := mergeFiles(listFileName)
	defer os.Remove(mergedFileName)
	t0 := getDuration(mergedFileName)
	episodes := splitAsync(book, t0, mergedFileName)

	sort.Sort(episodes)

	xmlPath := generateXML(book, episodes)
	m3uPath := generateM3U(book, episodes)
	infoLog.Println(xmlPath)
	infoLog.Println(m3uPath)
	return nil
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
	defer os.Remove(splitedFilename)
	simpleExec("ffmpeg", "-y", "-i", source, "-acodec", "copy", "-t", s1s, "-ss", t1s, splitedFilename)
	simpleExec("lame", "-V", "9", "--vbr-new", "-mm", "-h", "-q", "0", "-f", splitedFilename, fpath)
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

	if src == "" {
		log.Fatalln("No source found.")
	}

	pwd, err := os.Getwd()
	check(err)

	warningLog.Println("No destination specifyed. '" + pwd + "' used")
	if dst == "" {
		dst = pwd
	}

	if bookID == "" {
		bookID = filepath.Base(src)
		warningLog.Println("No name specifyed. '" + bookID + "' used")
	}

	dest := path.Join(dst, bookID)
	err = os.Mkdir(dest, 0777)
	check(err)

	book := bookMeta{
		id:  bookID,
		src: src,
		dst: dest,
	}
	err = cookAudio(book)
	check(err)
	infoLog.Println("Done")
}
