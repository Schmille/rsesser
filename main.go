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
	Numbers     bool
	OldestFirst bool
}

type NameOptions struct {
	Options
	FeedLength       int
	FeedCurrent      int
	EnclosureLength  int
	EnclosureCurrent int
	Url              string
	Filename         string
}

func main() {
	start := time.Now()

	options := Options{}
	flag.BoolVar(&options.Numbers, "n", false, "add numbers for track positions")
	flag.BoolVar(&options.OldestFirst, "l", false, "download oldest entries first")
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
	queue := feed.Items
	if opt.OldestFirst {
		queue = reverse(queue)
	}

	for index, item := range queue {
		zeroPadding := orderOfMagnitude(len(feed.Items)) + 1
		log.Printf("Starting item (%0*d / %d) %s\n", zeroPadding, index+1, len(feed.Items), item.Title)

		for encIndex, enclosure := range item.Enclosures {

			nameOpts := NameOptions{
				Options:          opt,
				FeedLength:       len(feed.Items),
				FeedCurrent:      index,
				EnclosureLength:  len(item.Enclosures),
				EnclosureCurrent: encIndex,
				Url:              enclosure.URL,
				Filename:         item.Title,
			}
			name := createFilename(nameOpts)

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

func createFilename(opt NameOptions) string {
	positionNamePart := ""
	if opt.Numbers {
		feedZeroPadding := orderOfMagnitude(opt.FeedLength) + 1

		position := 0
		if !opt.OldestFirst {
			position = opt.FeedLength - opt.FeedCurrent
		} else {
			position = opt.FeedCurrent + 1
		}

		positionNamePart = fmt.Sprintf("%0*d", feedZeroPadding, position)
		if opt.EnclosureLength > 1 {
			encZeroPadding := orderOfMagnitude(opt.EnclosureLength) + 1
			positionNamePart += fmt.Sprintf(".%0*d", encZeroPadding, opt.EnclosureLength-opt.EnclosureCurrent)
		}
		positionNamePart += " - "
	}

	ext := path.Ext(opt.Url)
	name := cleanFilename(opt.Filename) + ext
	name = positionNamePart + name
	return name
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

func reverse(items []*gofeed.Item) []*gofeed.Item {
	out := make([]*gofeed.Item, len(items))
	j := 0
	for i := len(items) - 1; i >= 0; i-- {
		out[j] = items[i]
		j++
	}
	return out
}
