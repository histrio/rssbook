package main

import (
	"strings"

	"bufio"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
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
	pos   int
	book  bookMeta
	skip  time.Time
	limit time.Time
}

type bookMeta struct {
	id       string
	src      string
	dst      string
	title    string
	author   string
	episodes episodesList
}

type bookEpisode struct {
	//book bookMeta
	file string
	pos  int
}

type episodesList []bookEpisode
type entrySorter []rssEntry

func (a entrySorter) Len() int           { return len(a) }
func (a entrySorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a entrySorter) Less(i, j int) bool { return a[i].ID < a[j].ID }

func (slice episodesList) Len() int {
	return len(slice)
}

func (slice episodesList) Less(i, j int) bool {
	return slice[i].pos < slice[j].pos
}

func (slice episodesList) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

const siteURL string = "https://books.falseprotagonist.me/"
const s3Url string = "https://s3-eu-west-1.amazonaws.com"
const s3Bucket string = "falseprotagonist-one"

const _defaultBookAuthor string = "< Book Author >"
const _defaultBookTitle string = "< Title >"

func splitAsync(book bookMeta, t0 time.Time, src string) episodesList {
	infoLog.Println("Spliting")
	book.src = src

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

	var wg2 sync.WaitGroup
	wg2.Add(1)

	go func() {
		defer wg2.Done()
		for t := range data {
			result = append(result, t)
		}
	}()

	pos := 0
	for t1.Before(t0) {
		pos = pos + 1
		task := splitterTask{
			pos:   pos,
			book:  book,
			skip:  t1,
			limit: s1,
		}
		tasks <- task
		t1 = t1.Add(time.Minute * 5)
	}

	for i := 0; i < jobs; i++ {
		tasks <- splitterTask{pos: -1}
	}
	close(tasks)
	wg.Wait()
	close(data)
	wg2.Wait()

	return result
}

func generateM3U(book bookMeta) string {
	m3uDest := path.Join(book.dst, book.id+".m3u")
	f, err := os.Create(m3uDest)
	check(err)
	f.WriteString("#EXTM3U\n\n")
	for _, ep := range book.episodes {
		path, err := filepath.Rel(book.dst, ep.file)
		check(err)
		f.WriteString(path + "\n")
	}
	return m3uDest
}

func cookAudio(book bookMeta) (result episodesList, err error) {
	listFileName := getFileListFile(book.src)
	defer os.Remove(listFileName)
	mergedFileName := mergeFiles(listFileName)
	defer os.Remove(mergedFileName)
	t0 := getDuration(mergedFileName)
	result = splitAsync(book, t0, mergedFileName)
	sort.Sort(result)
	return result, nil
}

func runner(bookID string, tasks chan splitterTask, data chan bookEpisode, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		task := <-tasks
		if task.pos == -1 {
			break
		}
		filename := split(task.book, task.skip, task.limit, task.pos)
		episode := bookEpisode{
			pos:  task.pos,
			file: filename,
		}
		data <- episode
	}
}

func getDefaultBookMeta(book bookMeta) (author string, title string) {

	first := getFileList(book.src)[0]
	metaFile, err := ioutil.TempFile(os.TempDir(), "prefix")
	defer metaFile.Close()
	check(err)
	simpleExec("ffmpeg", "-y", "-i", first, "-f", "ffmetadata", metaFile.Name())
	f, err := os.Open(metaFile.Name())
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.SplitN(scanner.Text(), "=", 2)
		if len(line) == 2 {
			attr, value := line[0], line[1]
			switch attr {
			case "artist":
				author = value
			case "album":
				title = value
			}
		}
	}
	return author, title
}

func main() {
	initLoggers(os.Stdout, os.Stdout, os.Stderr)
	var dst string
	var src string
	var bookID string
	var bookTitle string
	var bookAuthor string

	flag.StringVar(&dst, "dst", "", "Generated files destination")
	flag.StringVar(&src, "src", "", "Source of audiofiles")
	flag.StringVar(&bookID, "name", "", "Shortname")
	flag.StringVar(&bookTitle, "title", "", "Title")
	flag.Parse()

	if src == "" {
		log.Fatalln("No source found.")
	}

	pwd, err := os.Getwd()
	check(err)

	if dst == "" {
		dst = pwd
		warningLog.Println("No destination specifyed. '" + pwd + "' used")
	}

	if bookID == "" {
		bookID = filepath.Base(src)
		warningLog.Println("No book-id specifyed. '" + bookID + "' used")
	}

	dest := path.Join(dst, bookID)
	err = os.Mkdir(dest, 0777)
	check(err)

	book := bookMeta{
		id:  bookID,
		src: src,
		dst: dest,
	}

	defaultBookAuthor, defaultBookTitle := getDefaultBookMeta(book)

	if bookTitle == "" {
		bookTitle = defaultBookTitle
		warningLog.Println("No title specifyed. '" + bookTitle + "' used")
	}

	if bookAuthor == "" {
		bookAuthor = defaultBookAuthor
		warningLog.Println("No author specifyed. '" + bookAuthor + "' used")
	}

	book.title = bookTitle
	book.author = bookAuthor

	episodes, err := cookAudio(book)
	check(err)
	book.episodes = episodes
	xmlPath := generateXML(book)

	m3uPath := generateM3U(book)
	infoLog.Println(xmlPath)
	infoLog.Println(m3uPath)
	infoLog.Println("Done")
}
