package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	//"sort"
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
	name     string
	href     string
	n        string
	pos      int
	file     string
	filename string
	fileSize int64
	duration time.Duration
}

type episodesList []bookEpisode
type entrySorter []rssItem

func (a entrySorter) Len() int           { return len(a) }
func (a entrySorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a entrySorter) Less(i, j int) bool { return a[i].GUID.Value < a[j].GUID.Value }

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
	s1 := t1.Add(time.Minute * 10)

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
		t1 = t1.Add(time.Minute * 10)
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

//func cookAudio(book bookMeta) (result episodesList, err error) {
//listFileName := getFileListFile(book.src)
//defer os.Remove(listFileName)
//mergedFileName := mergeFiles(listFileName)
//defer os.Remove(mergedFileName)
//t0 := getDuration(mergedFileName)
//result = splitAsync(book, t0, mergedFileName)
//sort.Sort(result)
//return result, nil
//}

func runner(bookID string, tasks chan splitterTask, data chan bookEpisode, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		task := <-tasks
		if task.pos == -1 {
			break
		}
		filename, duration := doSplit(task.book, task.skip, task.limit, task.pos)
		episode := bookEpisode{
			pos:      task.pos,
			filename: filename,
			duration: duration,
		}
		data <- episode
	}
}

func getDefaultBookMeta(book bookMeta) (author string, title string) {

	first := getFileList(book.src)[0]
	metaFile, err := ioutil.TempFile(os.TempDir(), "prefix")
	defer metaFile.Close()
	defer os.Remove(metaFile.Name())
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

type inputFile struct {
	name string
	path string
	size int64
}

type fileSplit struct {
	inputFile inputFile
	from      time.Duration
	to        time.Duration
}

type splitEpisode struct {
	pos      int
	name     string
	splits   []fileSplit
	file     inputFile
	duration time.Duration
}

func getFiles(dir string) chan inputFile {
	c := make(chan inputFile)
	go func() {
		files, err := ioutil.ReadDir(dir)
		check(err)
		for _, f := range files {
			fname := f.Name()
			ext := filepath.Ext(fname)
			if ext == ".mp3" {
				c <- inputFile{path: path.Join(dir, fname)}
			}
		}
		close(c)
	}()
	return c
}

func getSplittedEpisodes(in <-chan inputFile, limitMin int) chan splitEpisode {
	episodeLimit := time.Duration(limitMin) * time.Minute
	plan := make(chan splitEpisode)
	pos := 1
	go func() {
		splits := []fileSplit{}
		debt := time.Duration(0)
		for f := range in {
			duration := getDuration(f.path)
			t0 := time.Duration(0)
			if debt > time.Duration(0) {
				if debt <= duration {
					splits = append(splits, fileSplit{inputFile: f, from: t0, to: t0 + debt})
					plan <- splitEpisode{pos: pos, splits: splits}
					pos = pos + 1
					splits = []fileSplit{}
					t0 = debt
					debt = time.Duration(0)
				}
				if debt > duration {
					splits = append(splits, fileSplit{
						inputFile: f, from: t0, to: duration})
					debt = debt - duration
					continue
				}
			}
			for (t0 + episodeLimit) < duration {
				splits = append(splits, fileSplit{inputFile: f, from: t0, to: t0 + episodeLimit})
				plan <- splitEpisode{pos: pos, splits: splits}
				pos = pos + 1
				splits = []fileSplit{}
				t0 = t0 + episodeLimit
			}
			splits = append(splits, fileSplit{inputFile: f, from: t0, to: duration})
			debt = episodeLimit - (duration - t0)
		}
		plan <- splitEpisode{pos: pos, splits: splits}
		close(plan)
	}()

	return plan
}

func getMergedEpisodes(in <-chan splitEpisode) chan splitEpisode {
	c := make(chan splitEpisode)
	go func() {
		for episode := range in {
			listFile, err := ioutil.TempFile(os.TempDir(), "rssbook_mergelist_")
			temp := []string{}
			check(err)
			for _, split := range episode.splits {
				tempFile, err := ioutil.TempFile(os.TempDir(), "rssbook_split_")
				check(err)
				name := tempFile.Name()
				temp = append(temp, name)
				simpleExec("ffmpeg", "-y", "-i", split.inputFile.path, "-acodec", "copy", "-f", "mp3",
					"-ss", formatDuration(split.from),
					"-to", formatDuration(split.to),
					"-write_xing", "0", name)
				listFile.WriteString(fmt.Sprintf("file '%v'\n", name))
			}
			listFile.Close()
			ep, err := ioutil.TempFile(os.TempDir(), "rssbook_concat_")
			check(err)
			simpleExec("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", listFile.Name(), "-f", "mp3", "-c", "copy", ep.Name())
			os.Remove(listFile.Name())

			go func() {
				for _, item := range temp {
					os.Remove(item)
				}
			}()

			episode.file = inputFile{
				path: ep.Name(),
			}
			c <- episode
		}
		close(c)
	}()
	return c
}

func getComressedEpisodes(book bookMeta, in <-chan splitEpisode) chan splitEpisode {
	c := make(chan splitEpisode)
	go func() {
		for ep := range in {
			listFile, err := ioutil.TempFile(os.TempDir(), "rssbook_compress_")
			check(err)

			albumMetadata := fmt.Sprintf("album=%v", book.title)
			authorMetadata := fmt.Sprintf("artist=%v", book.author)
			titleMetadata := fmt.Sprintf("title=%v", fmt.Sprintf("Episode%04d", ep.pos))
			trackMetadata := fmt.Sprintf("track=%d", ep.pos)

			simpleExec("ffmpeg", "-y", "-i", ep.file.path, "-metadata", trackMetadata, "-metadata", albumMetadata, "-metadata", titleMetadata, "-metadata", authorMetadata, "-codec:a", "libmp3lame", "-qscale:a", "9", "-f", "mp3", listFile.Name())
			os.Remove(ep.file.path)
			ep.file.path = listFile.Name()
			c <- ep
		}
		close(c)
	}()
	return c
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

	files := getFiles(src)
	splittedFiles := getSplittedEpisodes(files, 10)
	mergedEpisodes := getMergedEpisodes(splittedFiles)
	compressedEpisodes := getComressedEpisodes(book, mergedEpisodes)

	for ep := range compressedEpisodes {
		//_, filename := filepath.Split(ep.file.path)
		ep.file.size = getFileSize(ep.file.path)
		ep.duration = getDuration(ep.file.path)
		infoLog.Println(ep)
	}

	//episodes, err := cookAudio(book)
	//check(err)

	//updatedEpisodes := episodesList{}
	//for _, ep := range episodes {
	//ep.n = fmt.Sprintf("%04d", ep.pos)
	//_, ep.filename = filepath.Split(ep.file)
	//ep.name = fmt.Sprintf("Episode%s", ep.n)
	//ep.fileSize = getFileSize(ep.file)
	//ep.href = strings.Join([]string{s3Url, s3Bucket, book.id, ep.filename}, "/")
	//updatedEpisodes = append(updatedEpisodes, ep)
	//}

	//book.episodes = updatedEpisodes
	//xmlPath := generateXML(book)

	//m3uPath := generateM3U(book)
	//infoLog.Println(xmlPath)
	//infoLog.Println(m3uPath)
	//infoLog.Println("Done")
}
