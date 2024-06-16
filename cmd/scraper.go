package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

const (
	baseUrl    = "https://psnprofiles.com"
	guidesPage = baseUrl + "/guides/popular?page="
)

type GuideData struct {
	Game             string `json:"game"`
	Link             string `json:"link"`
	Difficulty       string `json:"difficulty"`
	TimeNeeded       string `json:"time_needed"`
	PlatinumRarity   string `json:"platinum_rarity"`
	Views            string `json:"views"`
	GuideRating      string `json:"guide_rating"`
	GuideRatingCount string `json:"guide_rating_count"`
	UserFavourites   string `json:"user_favourites"`
}

type Scraper struct {
	directory string
	formats   []string

	lastPage int

	collector *colly.Collector

	wg  sync.WaitGroup
	pmu sync.Mutex
	mu  sync.Mutex

	links chan string
	data  map[string]*GuideData
}

func NewScraper(directory string, formats []string) *Scraper {
	return &Scraper{
		directory: directory,
		formats:   formats,

		lastPage: 1,

		collector: colly.NewCollector(
			colly.Async(true),
			colly.AllowURLRevisit(),
			colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3"),
		),
		wg:    sync.WaitGroup{},
		pmu:   sync.Mutex{},
		mu:    sync.Mutex{},
		links: make(chan string),
		data:  make(map[string]*GuideData),
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
			s.handleGuideListPage(link, doc)
		}
	})
}

func (s *Scraper) scrape() {
	s.wg.Add(s.lastPage)
	go func() {
		for i := 1; i <= s.lastPage; i++ {
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
			fmt.Printf("unknown format: %s\n", format)
		}
	}

	s.wg.Wait()
}

func (s *Scraper) handleGuideListPage(link string, doc *goquery.Document) {
	doc.Find("a").Each(func(index int, item *goquery.Selection) {
		href, exists := item.Attr("href")
		if exists && strings.Contains(href, "/guide/") {
			s.wg.Add(1)
			s.links <- fmt.Sprintf("%s%s", baseUrl, href)
		}
	})
	s.pmu.Lock()
	maxLastPage := s.lastPage
	doc.Find(".pagination").First().Find("li").Each(func(index int, item *goquery.Selection) {
		_, hasClass := item.Attr("class")
		if !hasClass {
			page := item.Find("a").First().Text()
			pageInt, err := strconv.Atoi(page)
			if err != nil {
				fmt.Printf("page number could not be resolved from page %s\n", page)
			} else {
				if pageInt > maxLastPage {
					maxLastPage = pageInt
				}
			}
		}
	})
	// TODO once whole fetch logic is done, open this and fetch for long
	/*if maxLastPage > s.lastPage {
		fmt.Printf("found new max page: %d\n", maxLastPage)
		for i := s.lastPage + 1; i <= maxLastPage; i++ {
			s.wg.Add(1)
			s.links <- fmt.Sprintf("%s%d", guidesPage, i)
		}
		s.lastPage = maxLastPage
	}*/
	s.pmu.Unlock()

	fmt.Printf("page %s done\n", link)
	s.wg.Done()
}

func (s *Scraper) handleGuidePage(link string, doc *goquery.Document) {
	titleBar := doc.Find(".title-bar")
	game := titleBar.Find("h3:nth-of-type(1)").Find("a:nth-of-type(2)").Text()

	userFavourites := ""
	guideRating := ""
	guideRatingCount := ""
	views := ""
	doc.Find(".guide-info").Parent().Find("div:nth-of-type(2)").Children().Each(func(index int, item *goquery.Selection) {
		switch index {
		case 0:
			{
				userFavourites = item.Contents().First().Text()
			}
		case 1:
			{
				maxId := 0
				item.Children().First().Children().Each(func(index int, star *goquery.Selection) {
					if _, exists := star.Attr("checked"); exists {
						if id, idExists := star.Attr("id"); idExists {
							parts := strings.Split(id, "-")
							if len(parts) == 2 {
								idInt, err := strconv.Atoi(parts[1])
								if err == nil && maxId < idInt {
									maxId = idInt
								}
							}
						}
					}
				})

				foundRatingCount := 0
				ratingCount := strings.TrimSpace(item.Children().Last().Text())
				parts := strings.Split(ratingCount, " ")
				if len(parts) == 2 {
					ratingCountInt, err := strconv.Atoi(parts[0])
					if err == nil {
						foundRatingCount = ratingCountInt
					}
				}

				guideRatingCount = fmt.Sprint(foundRatingCount)
				if maxId > 0 || foundRatingCount > 0 {
					guideRating = fmt.Sprintf("%d/5", maxId)
				}
			}
		case 2:
			{
				views = item.Contents().First().Text()
			}
		default:
			{
			}
		}
	})

	overviewInfo := doc.Find(".overview-info")
	difficulty := overviewInfo.Find("span:nth-of-type(1)").Find("span:nth-of-type(1)").Text()
	timeNeeded := overviewInfo.Find("span:nth-of-type(3)").Find("span:nth-of-type(1)").Text()

	platinumInfo := doc.Find("img[alt='Platinum']").ParentsFiltered("tr").First()
	platinumRarity := platinumInfo.Children().Eq(platinumInfo.Children().Length() - 2).First().Children().First().Find("span").First().Text()
	if platinumRarity == "" {
		platinumRarity = "N/A"
	}

	s.mu.Lock()
	s.data[link] = &GuideData{
		Game:             game,
		Link:             link,
		Difficulty:       difficulty,
		TimeNeeded:       timeNeeded,
		PlatinumRarity:   platinumRarity,
		Views:            views,
		GuideRating:      guideRating,
		GuideRatingCount: guideRatingCount,
		UserFavourites:   userFavourites,
	}
	s.mu.Unlock()

	fmt.Printf("guide page %s done\n", link)

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
		{"game", "link", "difficulty", "time_needed", "platinum_rarity", "views", "guide_rating", "guide_rating_count", "user_favourites"},
	}
	for _, entry := range s.data {
		entries = append(entries, []string{
			entry.Game,
			entry.Link,
			entry.Difficulty,
			entry.TimeNeeded,
			entry.PlatinumRarity,
			entry.Views,
			entry.GuideRating,
			entry.GuideRatingCount,
			entry.UserFavourites,
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
	builder.WriteString("| **game** | **difficulty** | **time_needed** | **platinum_rarity** | **views** | **guide_rating** | **guide_rating_count ** | **user_favourites ** |\n")
	builder.WriteString("|:--------|:--------:|:-------:|:-----:|:------:|:------:|:-----:|:-----:|\n")

	for _, entry := range s.data {
		if entry.Game != "" {
			_, err = builder.WriteString(
				fmt.Sprintf(
					"| [%s](%s) | %s | %s | %s | %s | %s | %s | %s |\n",
					entry.Game,
					entry.Link,
					entry.Difficulty,
					entry.TimeNeeded,
					entry.PlatinumRarity,
					entry.Views,
					entry.GuideRating,
					entry.GuideRatingCount,
					entry.UserFavourites,
				),
			)
		} else {
			_, err = builder.WriteString(
				fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s |\n",
					entry.Link,
					entry.Difficulty,
					entry.TimeNeeded,
					entry.PlatinumRarity,
					entry.Views,
					entry.GuideRating,
					entry.GuideRatingCount,
					entry.UserFavourites,
				),
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
