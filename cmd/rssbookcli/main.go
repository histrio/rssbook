package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gosimple/slug"
	"github.com/histrio/rssbook/pkg/audio"
	"github.com/histrio/rssbook/pkg/loggers"
	"github.com/histrio/rssbook/pkg/rss"
	"github.com/histrio/rssbook/pkg/utils"
	"github.com/histrio/rssbook/pkg/version"
)

const _defaultBookAuthor string = "< Book Author >"
const _defaultBookTitle string = "< Title >"
const episodeMin int = 8

func getTitleAndAuthor(src string) (string, string) {
	firstFieldName := <-utils.GetFiles(src)
	result, err := utils.SimpleExec("ffprobe", "-loglevel", "error", "-show_entries", "format_tags=title,artist", "-of", "default=noprint_wrappers=1:nokey=1", "-of", "csv", string(firstFieldName))
	utils.Check(err)
	artistAndTitle := strings.Split(result, ",")
	if len(artistAndTitle) < 3 {
		return "", ""
	}
	return artistAndTitle[1], artistAndTitle[2]
}

func cookAudio(src string) chan utils.FileName {
	files := utils.GetFiles(src)
	splittedFiles := audio.GetSplittedEpisodes(files, episodeMin)
	mergedEpisodes := audio.GetMergedEpisodes(splittedFiles)
	compressedEpisodes := audio.GetCompressedEpisodes(mergedEpisodes)
	return compressedEpisodes
}

func cookRss(book utils.BookMeta, dst string) utils.FileName {
	xmlDest := path.Join(dst, book.ID+".xml")
	f, err := os.Create(xmlDest)
	utils.Check(err)
	defer f.Close()
	f.WriteString(rss.GenerateXML(book))
	return utils.FileName(xmlDest)
}

func cookM3U(book utils.BookMeta, dst string) string {
	m3uDest := path.Join(dst, book.ID+".m3u")
	f, err := os.Create(m3uDest)
	utils.Check(err)
	f.WriteString("#EXTM3U\n\n")
	for _, ep := range book.Episodes {
		f.WriteString(ep.File + ".mp3\n")
	}
	return m3uDest
}

func main() {
	loggers.InitLoggers(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	loggers.Info.Printf(
		"Starting ...\ncommit: %s, build time: %s, release: %s",
		version.Commit, version.BuildTime, version.Release,
	)
	var dst string
	var src string
	var bookID string
	var bookTitle string
	var bookAuthor string

	flag.StringVar(&dst, "dst", "", "Generated files destination")
	flag.StringVar(&src, "src", "", "Source of audiofiles")
	flag.StringVar(&bookID, "name", "", "Set a shortname for the podcast. By default it would be a slugifyed source folder name.")
	flag.StringVar(&bookTitle, "title", "", "Set title for the podcast. By default it would take a title from the first file of the book.")
	flag.StringVar(&bookAuthor, "author", "", "Set an author for the podcast. By default it would take an artist from the first file of the book.")
	flag.Parse()

	if src == "" {
		loggers.Warning.Fatalln("No source found.")
	}

	pwd, err := os.Getwd()
	utils.Check(err)

	if dst == "" {
		dst = pwd
		loggers.Warning.Println("No destination specified. '" + pwd + "' used")
	}

	if bookID == "" {
		bookID = slug.Make(filepath.Base(src))
		loggers.Warning.Println("No book-id specified. '" + bookID + "' used")
	}

	dest := path.Join(dst, bookID)
	err = os.Mkdir(dest, 0777)
	utils.Check(err)

	_title, _author := getTitleAndAuthor(src)
	if bookAuthor == "" {
		bookAuthor = _author
		loggers.Warning.Println("No book author specified. '" + bookAuthor + "' used")
	}
	if bookTitle == "" {
		bookTitle = _title
		loggers.Warning.Println("No book author specified. '" + bookTitle + "' used")
	}

	book := utils.BookMeta{
		ID:     bookID,
		Title:  bookTitle,
		Author: bookAuthor,
	}

	pos := 0
	for epFile := range cookAudio(src) {
		loggers.Info.Println("Issued: " + epFile)

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
			Href:     utils.S3Url + book.ID + "/" + filename + ".mp3",
			Duration: audio.GetDuration(epFile),
		}

		book.Episodes = append(book.Episodes, ep)
	}

	cookRss(book, dest)
	cookM3U(book, dest)
}
