package main

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

type rssEnclosure struct {
	Url    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

type rssItemGUID struct {
	IsPermaLink bool   `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type rssItem struct {
	Title       string       `xml:"title"`
	Link        string       `xml:"link"`
	Description string       `xml:"description"`
	Author      string       `xml:"author,omitempty"`
	Category    string       `xml:"category,omitempty"`
	Comments    string       `xml:"comments,omitempty"`
	GUID        rssItemGUID  `xml:"guid"`
	Enclosure   rssEnclosure `xml:"enclosure"`
	PubDate     time.Time    `xml:"pubDate"`
	Source      string       `xml:"source"`

	ItunesDuration string `xml:"itunes:duration"`
	ItunesExplicit string `xml:"itunes:explicit"`
}

type rssBody struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Content string     `xml:"xmlns:content,attr"`
	Atom    string     `xml:"xmlns:atom,attr"`
	Itunes  string     `xml:"xmlns:itunes,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssAtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssItunesOwner struct {
	XMLName xml.Name `xml:"itunes:owner,omitempty"`
	Name    string   `xml:"itunes:name"`
	Email   string   `xml:"itunes:email"`
}

type rssCloud struct {
	XMLName           xml.Name `xml:"cloud,omitempty"`
	Domain            string   `xml:"domain,attr"`
	Port              string   `xml:"port,attr"`
	Path              string   `xml:"path,attr"`
	RegisterProcedure string   `xml:"registerProcedure,attr"`
	Protocol          string   `xml:"protocol,attr"`
}

type rssItunesCategory struct {
	XMLName xml.Name `xml:"itunes:category"`
	Text    string   `xml:"text,attr"`
}

type rssChannel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`

	Image          rssImage  `xml:"image,omitempty"`
	ManagingEditor string    `xml:"managingEditor,omitempty"`
	Language       string    `xml:"language"`
	Copyrigt       string    `xml:"copyrigt,omitempty"`
	LastBuildDate  time.Time `xml:"lastBuildDate,omitempty"`
	Docs           string    `xml:"docs,omitempty"`
	TTL            string    `xml:"ttl,omitempty"`
	WebMaster      string    `xml:"webMaster,omitempty"`
	Category       string    `xml:"category,omitempty"`
	Generator      string    `xml:"generator,omitempty"`
	Cloud          *rssCloud `xml:"cloud,omitempty"`
	Rating         string    `xml:"rating,omitempty"`

	AtomLink       rssAtomLink `xml:"atom:link,omitempty"`
	ItunesOwner    *rssItunesOwner
	ItunesCategory *rssItunesCategory
	ItunesExplicit string `xml:"itunes:explicit"`

	Entries []rssItem `xml:"item"`
}

type rssImage struct {
	Title  string `xml:"title"`
	Link   string `xml:"link"`
	Url    string `xml:"url"`
	Width  string `xml:"width"`
	Height string `xml:"height"`
}

func generateXML(book bookMeta) string {

	items := []rssItem{}
	t0 := time.Now()
	for _, ep := range book.episodes {
		item := rssItem{
			Title: ep.name,
			Link:  ep.href,
			GUID: rssItemGUID{
				IsPermaLink: false,
				Value:       getid("books.falseprotagonist.me", fmt.Sprintf("%s%s", book.id, ep.n), t0),
			},
			Enclosure: rssEnclosure{
				Url:    ep.href,
				Type:   "audio/mpeg",
				Length: ep.fileSize,
			},
			PubDate:        t0.Add(time.Second * time.Duration(ep.pos)),
			ItunesExplicit: "no",
			//ItunesDuration: ep.duration.Format("15:04:05"),
		}
		items = append(items, item)
	}

	selfLink := strings.Join([]string{s3Url, s3Bucket, book.id + ".xml"}, "/")
	rss := &rssBody{
		Version: "2.0",
		Content: "http://purl.org/rss/1.0/modules/content/",
		Atom:    "http://www.w3.org/2005/Atom",
		Itunes:  "http://www.itunes.com/dtds/podcast-1.0.dtd",
		Channel: rssChannel{
			Title:       book.title,
			Link:        selfLink,
			Description: "Audiobook as a podcast",
			Language:    "ru",
			Entries:     items,
			Docs:        "http://blogs.law.harvard.edu/tech/rss",
			AtomLink: rssAtomLink{
				Href: selfLink,
				Rel:  "self",
				Type: "application/rss+xml",
			},
			LastBuildDate: t0,
			Image: rssImage{
				Title:  book.title,
				Link:   "https://falseprotagonist.me",
				Url:    "https://s3-eu-west-1.amazonaws.com/falseprotagonist-one/audiobook.png",
				Width:  "144",
				Height: "144",
			},
			ItunesExplicit: "no",
			ItunesCategory: &rssItunesCategory{
				Text: "Education",
			},
		},
	}

	out, err := xml.MarshalIndent(rss, "", "  ")
	check(err)
	return xml.Header + string(out)
}
