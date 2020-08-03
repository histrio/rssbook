package main

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"testing"

	"github.com/histrio/rssbook/pkg/rss"
	"github.com/histrio/rssbook/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func Test_cookRss(t *testing.T) {
	dir, err := ioutil.TempDir("", "rssbook")
	if err != nil {
		log.Fatal(err)
	}

	book := utils.BookMeta{ID: "test"}
	result := cookRss(book, dir)
	assert.Equal(t, result, utils.FileName(dir+"/test.xml"))

	data, err := ioutil.ReadFile(string(result))
	if err != nil {
		log.Fatal(err)
	}
	var xmlData rss.RssBody
	xml.Unmarshal(data, &xmlData)
}

func Test_cookM3U(t *testing.T) {
	dir, err := ioutil.TempDir("", "rssbook")
	if err != nil {
		log.Fatal(err)
	}

	book := utils.BookMeta{ID: "test"}
	result := cookM3U(book, dir)
	data, err := ioutil.ReadFile(string(result))
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(t, string(data), "#EXTM3U\n\n")
}
