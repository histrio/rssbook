package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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
	XMLName   xml.Name  `xml:"http://www.w3.org/2005/Atom rss"`
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

func generateXML(book bookMeta) string {
	infoLog.Println("Generating xml")
	entries := []rssEntry{}
	t0 := time.Now()
	for _, ep := range book.episodes {
		n := fmt.Sprintf("%04d", ep.pos)
		_, epFilename := filepath.Split(ep.file)
		epName := fmt.Sprintf("Episode%s", n)
		//epSize := getFileSize(ep.file)

		content := fmt.Sprintf("Episode %s for %s", n, book.id)
		href := strings.Join([]string{s3Url, s3Bucket, book.id, epFilename}, "/")
		entry := rssEntry{
			Title:   epName,
			ID:      getid("books.falseprotagonist.me", fmt.Sprintf("%s%s", book.id, n), t0),
			Updated: t0.Add(time.Second * time.Duration(ep.pos)),
			LinkList: []rssLink{
				rssLink{Href: siteURL + book.id, Rel: "alternate"},
				rssLink{
					Href:  href,
					Rel:   "alternate",
					Type:  "audio/mpeg",
					Title: epName,
					//Length: epSize,
				},
			},
			Author: rssAuthor{
				Name:  book.author,
				Email: "rh@rh.rh",
			},
			Content: content,
		}
		entries = append(entries, entry)
	}

	sort.Sort(entrySorter(entries))

	selfLink := strings.Join([]string{s3Url, s3Bucket, book.id + ".xml"}, "/")
	rss := &rssBody{
		Title:    book.title,
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
	f.WriteString(xml.Header + string(out))
	return xmlDest
}
