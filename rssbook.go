package main

import (
	"encoding/xml"
	"fmt"
	"time"
)

type Link struct {
	Href   string `xml:"href,attr"`
	Rel    string `xml:"rel,attr"`
	Type   string `xml:"type,attr,omitempty"`
	Title  string `xml:"title,attr,omitempty"`
	Length int    `xml:"length,attr"`
}

type Entry struct {
	Title    string    `xml:"title"`
	Id       string    `xml:"id"`
	Updated  time.Time `xml:"updated"`
	LinkList []Link    `xml:"link"`
	Author   Author    `xml:"author"`
	Content  string    `xml:"content"`
}

type Author struct {
	Name  string `xml:"name"`
	Email string `xml:"email"`
}

type Atom1 struct {
	XMLName   xml.Name  `xml:"http://www.w3.org/2005/Atom feed"`
	Title     string    `xml:"title"`
	Author    Author    `xml:"author,omitempty"`
	Id        string    `xml:"id"`
	Subtitle  string    `xml:"subtitle"`
	LinkList  []Link    `xml:"link"`
	Generator string    `xml:"generator"`
	Updated   time.Time `xml:"updated"`
	//Rights   string   `xml:"rights"`
	EntryList []Entry `xml:"entry"`
}

func getid(domain string, link string, date time.Time) string {
	date_formatted := fmt.Sprintf("%d-%02d-%02d", date.Year(), date.Month(), date.Day())
	return fmt.Sprintf("tag:%v,%v:%v", domain, date_formatted, link)
}

func main() {

	entry := Entry{
		Title:   "Episode1",
		Id:      getid("books.falseprotagonist.me", "/readyplayerone", time.Now()),
		Updated: time.Now(),
		LinkList: []Link{
			Link{Href: "https://falseprotagonist.me", Rel: "alternate"},
			Link{
				Href:   "https://falseprotagonist.me/test.mp3",
				Rel:    "alternate",
				Type:   "audio/mpeg",
				Title:  "MP3",
				Length: 1234,
			},
		},
		Author: Author{
			Name:  "Robert Harrison",
			Email: "rh@rh.rh",
		},
		Content: "test",
	}

	rss := &Atom1{
		Title:    "Ready Player One (Book)",
		Id:       getid("books.falseprotagonist.me", "/readyplayerone", time.Now()),
		Subtitle: "Audiobook as a podcast",
		LinkList: []Link{
			Link{Href: "https://falseprotagonist.me", Rel: "self"},
		},
		Updated:   time.Now(),
		Generator: "rssbook/0.1(+https://github.com/histrio/rssbook)",
		EntryList: []Entry{
			entry,
		},
	}

	out, err := xml.MarshalIndent(rss, "", "  ")

	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))
}
