package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"time"
)

type rssEntry struct {
	ID string `xml:"id"`
}

type rssBody struct {
	XMLName xml.Name   `xml:"rss"`
	Version int        `xml:"version,attr"`
	Channel []rssEntry `xml:"channel"`
}

func generateXML(book bookMeta) string {

	entries := []rssEntry{}
	t0 := time.Now()
	for _, ep := range book.episodes {
		n := fmt.Sprintf("%04d", ep.pos)
		entry := rssEntry{
			ID: getid("books.falseprotagonist.me", fmt.Sprintf("%s%s", book.id, n), t0),
		}
		entries = append(entries, entry)

	}

	rss := &rssBody{
		Version: 2,
		Channel: entries,
	}
	out, err := xml.MarshalIndent(rss, "", "  ")
	check(err)
	xmlDest := path.Join(book.dst, book.id+".xml")
	f, err := os.Create(xmlDest)
	check(err)
	f.WriteString(xml.Header + string(out))
	return xmlDest
}
