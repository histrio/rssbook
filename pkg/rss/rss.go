package rss

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/histrio/rssbook/pkg/utils"
)

type RFC822Time struct {
	time.Time
}

func (t RFC822Time) MarshalText() ([]byte, error) {
	text := t.Time.Format("Mon, 02 Jan 2006 03:04:05 -0700")
	return []byte(text), nil
}

type Duration struct {
	time.Duration
}

func (d Duration) MarshalText() ([]byte, error) {
	d2 := d.Round(time.Second)
	h := d2 / time.Hour
	d2 -= h * time.Hour
	m := d2 / time.Minute
	d2 -= m * time.Minute
	s := d2 / time.Second
	text := fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	return []byte(text), nil
}

type rssEnclosure struct {
	URL    string `xml:"url,attr"`
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
	PubDate     RFC822Time   `xml:"pubDate"`
	Source      string       `xml:"source,omitempty"`

	ItunesDuration Duration `xml:"itunes:duration"`
	ItunesExplicit string   `xml:"itunes:explicit"`
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

	Image          rssImage   `xml:"image,omitempty"`
	ManagingEditor string     `xml:"managingEditor,omitempty"`
	Language       string     `xml:"language"`
	Copyright      string     `xml:"copyright,omitempty"`
	LastBuildDate  RFC822Time `xml:"lastBuildDate,omitempty"`
	Docs           string     `xml:"docs,omitempty"`
	TTL            string     `xml:"ttl,omitempty"`
	WebMaster      string     `xml:"webMaster,omitempty"`
	Category       string     `xml:"category,omitempty"`
	Generator      string     `xml:"generator,omitempty"`
	Cloud          *rssCloud  `xml:"cloud,omitempty"`
	Rating         string     `xml:"rating,omitempty"`

	AtomLink       rssAtomLink `xml:"atom:link,omitempty"`
	ItunesOwner    *rssItunesOwner
	ItunesCategory *rssItunesCategory
	ItunesExplicit string `xml:"itunes:explicit"`

	Entries []rssItem `xml:"item"`
}

type rssImage struct {
	Title  string `xml:"title"`
	Link   string `xml:"link"`
	URL    string `xml:"url"`
	Width  int    `xml:"width"`
	Height int    `xml:"height"`
}

func GenerateXML(book utils.BookMeta) string {

	items := []rssItem{}
	t0 := time.Now()
	for _, ep := range book.Episodes {
		item := rssItem{
			Title: ep.Name,
			Link:  ep.Href,
			GUID: rssItemGUID{
				IsPermaLink: false,
				Value:       utils.GetID("books.falseprotagonist.me", fmt.Sprintf("%s%d", book.Id, ep.Pos), t0),
			},
			Enclosure: rssEnclosure{
				URL:    ep.Href,
				Type:   "audio/mpeg",
				Length: ep.FileSize,
			},
			PubDate:        RFC822Time{t0.Add(time.Second * time.Duration(ep.Pos))},
			ItunesExplicit: "no",
			ItunesDuration: Duration{ep.Duration},
		}
		items = append(items, item)
	}

	selfLink := strings.Join([]string{utils.S3Url, book.Id + "/", book.Id + ".xml"}, "")
	bookHash := utils.GetMD5Hash(selfLink)
	imageSize := 1400
	imageURL := fmt.Sprintf("https://www.gravatar.com/avatar/%s?s=%d&d=retro&r=g", bookHash, imageSize)
	rss := &rssBody{
		Version: "2.0",
		Content: "http://purl.org/rss/1.0/modules/content/",
		Atom:    "http://www.w3.org/2005/Atom",
		Itunes:  "http://www.itunes.com/dtds/podcast-1.0.dtd",
		Channel: rssChannel{
			Title:       book.Title,
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
			LastBuildDate: RFC822Time{t0},
			Image: rssImage{
				Title:  book.Title,
				Link:   selfLink,
				URL:    imageURL,
				Width:  imageSize,
				Height: imageSize,
			},
			ItunesExplicit: "no",
			ItunesCategory: &rssItunesCategory{
				Text: "Education",
			},
		},
	}

	out, err := xml.MarshalIndent(rss, "", "  ")
	utils.Check(err)
	return xml.Header + string(out)
}
