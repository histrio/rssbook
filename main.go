package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
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

type bookMeta struct {
	id       string
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
	fileSize int64
	duration time.Duration
}

type episodesList []bookEpisode

const siteURL string = "https://www.falseprotagonist.me/"
const s3Url string = "http://files.falseprotagonist.me/"
const s3Bucket string = "falseprotagonist-one/"

const _defaultBookAuthor string = "< Book Author >"
const _defaultBookTitle string = "< Title >"

type fileSplit struct {
	inputFile fileName
	from      time.Duration
	to        time.Duration
}

type audioMeta struct {
	author string
	title  string
}

type splitPlan []fileSplit
type fileName string

func cookAudio(src string) chan fileName {
	files := getFiles(src)
	splittedFiles := getSplittedEpisodes(files, 10)
	mergedEpisodes := getMergedEpisodes(splittedFiles)
	compressedEpisodes := getComressedEpisodes(mergedEpisodes)
	return compressedEpisodes
}

func cookRss(wg *sync.WaitGroup, book bookMeta, dst string) fileName {
	defer wg.Done()
	xmlDest := path.Join(dst, book.id+".xml")
	f, err := os.Create(xmlDest)
	check(err)
	defer f.Close()
	f.WriteString(generateXML(book))
	return fileName(xmlDest)
}

func generateM3U(wg *sync.WaitGroup, book bookMeta, dst string) string {
	defer wg.Done()
	m3uDest := path.Join(dst, book.id+".m3u")
	f, err := os.Create(m3uDest)
	check(err)
	f.WriteString("#EXTM3U\n\n")
	for _, ep := range book.episodes {
		f.WriteString(ep.file + ".mp3\n")
	}
	return m3uDest
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
	flag.StringVar(&bookAuthor, "author", "", "Author")
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
		id:     bookID,
		title:  bookTitle,
		author: bookAuthor,
	}

	pos := 0
	wg := &sync.WaitGroup{}
	for epFile := range cookAudio(src) {
		infoLog.Println("Issued: " + epFile)

		pos = pos + 1

		_, filename := filepath.Split(string(epFile))

		go func() {
			copyFile(epFile, path.Join(dest, filename+".mp3"))
			check(err)
		}()

		ep := bookEpisode{
			pos:      pos,
			name:     filename,
			file:     filename,
			fileSize: getFileSize(epFile),
			href:     s3Url + book.id + "/" + filename + ".mp3",
			duration: getDuration(epFile),
		}

		book.episodes = append(book.episodes, ep)
		wg.Add(2)
		go cookRss(wg, book, dest)
		go generateM3U(wg, book, dest)
	}

	wg.Wait()
}
