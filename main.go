package main

import (
	"errors"
	"github.com/mmcdole/gofeed"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
)

func main() {
	url := os.Args[1]
	const filepath = "feed.xml"
	feedValue := fetchXMLDefinition(url)
	UpdateLocalFeedFile(filepath, feedValue)

	feed := readFeed(feedValue)
	Download(feed)
}

func Download(feed *gofeed.Feed) {
	for _, item := range feed.Items {
		log.Printf("Starting %s\n", item.Title)
		for _, enclosure := range item.Enclosures {
			ext := path.Ext(enclosure.URL)
			name := cleanFilename(item.Title) + ext

			if fileExists(name) {
				stats, err := os.Stat(name)
				if err != nil {
					continue
				}

				enclosureLength, err := strconv.ParseInt(enclosure.Length, 10, 64)
				if err != nil {
					continue
				}

				if stats.Size() < enclosureLength {
					os.Remove(name)
					continue
				}
			}

			resp, err := http.Get(enclosure.URL)
			if err != nil {
				continue
			}

			bytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				continue
			}

			ioutil.WriteFile(name, bytes, 777)
		}
	}
}

func UpdateLocalFeedFile(filepath string, feedValue string) {
	if !fileExists(filepath) {
		err := ioutil.WriteFile(filepath, []byte(feedValue), 777)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		fileBytes, err := ioutil.ReadFile(filepath)
		if err != nil {
			log.Fatalln(err)
		}
		content := string(fileBytes)
		if feedValue != content {
			err = os.Remove(filepath)
			if err != nil {
				log.Fatalln(err)
			}
			err = ioutil.WriteFile(filepath, []byte(feedValue), 777)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func fetchXMLDefinition(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	return string(bytes)
}

func readFeed(feedStr string) *gofeed.Feed {
	parser := gofeed.NewParser()
	feed, err := parser.ParseString(feedStr)
	if err != nil {
		log.Fatalln(err)
	}
	return feed
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func cleanFilename(name string) string {
	if runtime.GOOS == "windows" {
		name = strings.ReplaceAll(name, "\\", " - ")
		name = strings.ReplaceAll(name, "/", " - ")
		name = strings.ReplaceAll(name, ":", " - ")
		name = strings.ReplaceAll(name, "*", " - ")
		name = strings.ReplaceAll(name, "<", " - ")
		name = strings.ReplaceAll(name, ">", " - ")
		name = strings.ReplaceAll(name, "|", " - ")
		name = strings.ReplaceAll(name, "\"", " - ")
	}
	return name
}
