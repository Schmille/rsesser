package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/mmcdole/gofeed"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Options struct {
	Numbers bool
}

func main() {
	start := time.Now()

	options := Options{}
	flag.BoolVar(&options.Numbers, "n", true, "add numbers for track positions")
	flag.Parse()
	url := flag.Arg(0)

	const filepath = "feed.xml"
	feedValue := fetchXMLDefinition(url)
	UpdateLocalFeedFile(filepath, feedValue)

	feed := readFeed(feedValue)
	fmt.Printf("Found %d items!\n", len(feed.Items))
	Download(feed, options)

	stop := time.Now()
	fmt.Printf("Started at %s, stopped at %s, took %s", start, stop, stop.Sub(start))
}

func Download(feed *gofeed.Feed, opt Options) {
	for index, item := range feed.Items {
		zeroPadding := orderOfMagnitude(len(feed.Items)) + 1
		log.Printf("Starting item (%0*d / %d) %s\n", zeroPadding, index+1, len(feed.Items), item.Title)

		for encIndex, enclosure := range item.Enclosures {

			position := ""
			if opt.Numbers {
				position = fmt.Sprintf("%0*d", zeroPadding, len(feed.Items)-index)
				if len(item.Enclosures) > 1 {
					position += fmt.Sprintf(".%0*d", orderOfMagnitude(len(item.Enclosures))+1, len(item.Enclosures)-encIndex)
				}
				position += " - "
			}

			ext := path.Ext(enclosure.URL)
			name := cleanFilename(item.Title) + ext
			name = position + name

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
				} else {
					log.Printf("File %s already exists and appears to be intact. Skipping...", name)
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

func orderOfMagnitude(number int) int {
	return int(math.Floor(math.Log10(float64(number))))
}
