package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
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
	id string
	//src      string
	//dst      string
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

const siteURL string = "https://books.falseprotagonist.me/"
const s3Url string = "https://s3-eu-west-1.amazonaws.com"
const s3Bucket string = "falseprotagonist-one"

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

func cookRss(book bookMeta, dst string) fileName {
	xmlDest := path.Join(dst, book.id+".xml")
	f, err := os.Create(xmlDest)
	check(err)
	defer f.Close()
	f.WriteString(generateXML(book))
	return fileName(xmlDest)
}

func generateM3U(book bookMeta, dst string) string {
	m3uDest := path.Join(dst, book.id+".m3u")
	f, err := os.Create(m3uDest)
	check(err)
	f.WriteString("#EXTM3U\n\n")
	for _, ep := range book.episodes {
		f.WriteString(ep.file + "\n")
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

	for epFile := range cookAudio(src) {

		_, filename := filepath.Split(string(epFile))

		go func() {
			copyFile(epFile, path.Join(dest, filename))
			check(err)
		}()

		ep := bookEpisode{
			file: filename,
		}

		book.episodes = append(book.episodes, ep)
		go cookRss(book, dest)
		go generateM3U(book, dest)

	}
}
