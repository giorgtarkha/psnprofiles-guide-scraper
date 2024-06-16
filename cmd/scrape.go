package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

const (
	baseUrl    = "https://psnprofiles.com"
	guidesPage = baseUrl + "/guides?page="
)

type GuideData struct {
	Game       string `json:"game"`
	Link       string `json:"link"`
	Difficulty string `json:"difficulty"`
	TimeNeeded string `json:"time_needed"`
}

type Scraper struct {
	directory string
	formats   []string

	collector *colly.Collector

	wg sync.WaitGroup
	mu sync.Mutex

	links chan string
	data  []*GuideData
}

func NewScraper(directory string, formats []string) *Scraper {
	return &Scraper{
		directory: directory,
		formats:   formats,
		collector: colly.NewCollector(
			colly.Async(true),
			colly.AllowURLRevisit(),
		),
		wg:    sync.WaitGroup{},
		mu:    sync.Mutex{},
		links: make(chan string),
		data:  []*GuideData{},
	}
}

func (s *Scraper) init() {
	s.collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5,
		Delay:       5 * time.Second,
	})
	s.collector.OnError(func(r *colly.Response, err error) {
		fmt.Printf("failed on link %s, requeuing link: %s\n", r.Request.URL.String(), err.Error())
		s.links <- r.Request.URL.String()
	})
	s.collector.OnResponse(func(r *colly.Response) {
		link := r.Request.URL.String()
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(r.Body))
		if err != nil {
			fmt.Printf("failed to parse html for link %s: %s\n", link, err.Error())
			s.links <- r.Request.URL.String()
			return
		}

		if strings.Contains(link, "/guide/") {
			s.handleGuidePage(link, doc)
		} else {
			s.handleGuideListPage(doc)
		}
	})
}

func (s *Scraper) scrape() {
	s.wg.Add(1)
	go func() {
		for i := 1; i <= 1; i++ {
			s.links <- fmt.Sprintf("%s%d", guidesPage, i)
		}
	}()

	go func() {
		for link := range s.links {
			fmt.Printf("visiting %s\n", link)
			go s.collector.Visit(link)
		}
	}()

	s.wg.Wait()
	close(s.links)

	for _, format := range s.formats {
		switch format {
		case FORMAT_JSON:
			{
				s.wg.Add(1)
				go func() {
					if err := s.dumpJson(); err != nil {
						fmt.Printf("failed to export to json: %s\n", err.Error())
					}
					s.wg.Done()
				}()
			}
		case FORMAT_CSV:
			{
				s.wg.Add(1)
				go func() {
					if err := s.dumpCsv(); err != nil {
						fmt.Printf("failed to export to csv: %s\n", err.Error())
					}
					s.wg.Done()
				}()
			}
		case FORMAT_MD:
			{
				s.wg.Add(1)
				go func() {
					if err := s.dumpMd(); err != nil {
						fmt.Printf("failed to export to md: %s\n", err.Error())
					}
					s.wg.Done()
				}()
			}
		default:
			fmt.Printf("unknown format: [%s]", format)
		}
	}

	s.wg.Wait()
}

func (s *Scraper) handleGuideListPage(doc *goquery.Document) {
	doc.Find("a").Each(func(index int, item *goquery.Selection) {
		href, exists := item.Attr("href")
		if exists && strings.Contains(href, "/guide/") {
			s.wg.Add(1)
			s.links <- fmt.Sprintf("%s%s", baseUrl, href)
		}
	})
	s.wg.Done()
}

func (s *Scraper) handleGuidePage(link string, doc *goquery.Document) {
	titleBar := doc.Find(".title-bar")
	game := titleBar.Find("h3:nth-of-type(1)").Find("a:nth-of-type(2)").Text()

	overviewInfo := doc.Find(".overview-info")
	difficulty := overviewInfo.Find("span:nth-of-type(1)").Find("span:nth-of-type(1)").Text()
	timeNeeded := overviewInfo.Find("span:nth-of-type(3)").Find("span:nth-of-type(1)").Text()

	s.mu.Lock()
	s.data = append(s.data, &GuideData{
		Game:       game,
		Link:       link,
		Difficulty: difficulty,
		TimeNeeded: timeNeeded,
	})
	s.mu.Unlock()

	s.wg.Done()
}

func (s *Scraper) dumpJson() error {
	file, err := os.Create(filepath.Join(s.directory, "guide_data.json"))
	if err != nil {
		return err
	}
	defer file.Close()

	jsonData, err := json.MarshalIndent(s.data, "", " ")
	if err != nil {
		return err
	}

	if _, err := file.Write(jsonData); err != nil {
		return err
	}

	return nil
}

func (s *Scraper) dumpCsv() error {
	file, err := os.Create(filepath.Join(s.directory, "guide_data.csv"))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	entries := [][]string{
		{"game", "link", "difficulty", "time_needed"},
	}
	for _, entry := range s.data {
		entries = append(entries, []string{
			entry.Game,
			entry.Link,
			entry.Difficulty,
			entry.TimeNeeded,
		})
	}

	if err := writer.WriteAll(entries); err != nil {
		return err
	}

	return nil
}

func (s *Scraper) dumpMd() error {
	file, err := os.Create(filepath.Join(s.directory, "guide_data.md"))
	if err != nil {
		return err
	}
	defer file.Close()

	builder := strings.Builder{}
	builder.WriteString("| **game** | **difficulty** | **time_needed** |\n")
	builder.WriteString("|:--------|:--------:|:-------:\n")

	for _, entry := range s.data {
		if entry.Game != "" {
			_, err = builder.WriteString(
				fmt.Sprintf("| (%s)[%s] | %s | %s |\n", entry.Game, entry.Link, entry.Difficulty, entry.TimeNeeded),
			)
		} else {
			_, err = builder.WriteString(
				fmt.Sprintf("| %s | %s | %s |\n", entry.Link, entry.Difficulty, entry.TimeNeeded),
			)
		}
		if err != nil {
			return err
		}
	}

	if _, err = file.WriteString(builder.String()); err != nil {
		return err
	}

	return nil
}
