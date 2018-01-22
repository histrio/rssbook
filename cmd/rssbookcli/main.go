package main

import (
	"flag"
	"fmt"
	"github.com/histrio/rssbook/pkg/audio"
	"github.com/histrio/rssbook/pkg/rss"
	"github.com/histrio/rssbook/pkg/utils"
	"github.com/histrio/rssbook/pkg/version"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
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

const _defaultBookAuthor string = "< Book Author >"
const _defaultBookTitle string = "< Title >"

func cookAudio(src string) chan utils.FileName {
	files := utils.GetFiles(src)
	splittedFiles := audio.GetSplittedEpisodes(files, 10)
	mergedEpisodes := audio.GetMergedEpisodes(splittedFiles)
	compressedEpisodes := audio.GetCompressedEpisodes(mergedEpisodes)
	return compressedEpisodes
}

func cookRss(wg *sync.WaitGroup, book utils.BookMeta, dst string) utils.FileName {
	defer wg.Done()
	xmlDest := path.Join(dst, book.Id+".xml")
	f, err := os.Create(xmlDest)
	utils.Check(err)
	defer f.Close()
	f.WriteString(rss.GenerateXML(book))
	return utils.FileName(xmlDest)
}

func generateM3U(wg *sync.WaitGroup, book utils.BookMeta, dst string) string {
	defer wg.Done()
	m3uDest := path.Join(dst, book.Id+".m3u")
	f, err := os.Create(m3uDest)
	utils.Check(err)
	f.WriteString("#EXTM3U\n\n")
	for _, ep := range book.Episodes {
		f.WriteString(ep.File + ".mp3\n")
	}
	return m3uDest
}

func main() {
	initLoggers(os.Stdout, os.Stdout, os.Stderr)
	infoLog.Printf(
		"Starting the service...\ncommit: %s, build time: %s, release: %s",
		version.Commit, version.BuildTime, version.Release,
	)
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
	utils.Check(err)

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
	utils.Check(err)

	book := utils.BookMeta{
		Id:     bookID,
		Title:  bookTitle,
		Author: bookAuthor,
	}

	pos := 0
	wg := &sync.WaitGroup{}
	for epFile := range cookAudio(src) {
		infoLog.Println("Issued: " + epFile)

		pos = pos + 1

		_, filename := filepath.Split(string(epFile))

		go func() {
			utils.CopyFile(epFile, path.Join(dest, filename+".mp3"))
			utils.Check(err)
		}()

		ep := utils.BookEpisode{
			Pos:      pos,
			Name:     fmt.Sprintf("Episode %03d", pos),
			File:     filename,
			FileSize: utils.GetFileSize(epFile),
			Href:     utils.S3Url + book.Id + "/" + filename + ".mp3",
			Duration: audio.GetDuration(epFile),
		}

		book.Episodes = append(book.Episodes, ep)
		wg.Add(2)
		go cookRss(wg, book, dest)
		go generateM3U(wg, book, dest)
	}

	wg.Wait()
}
