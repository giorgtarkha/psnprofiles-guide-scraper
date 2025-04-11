package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
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
	Platforms        string `json:"platforms"`
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
	sortings  []*Sorting

	lastPage int

	collector *colly.Collector

	wg  sync.WaitGroup
	pmu sync.Mutex
	mu  sync.Mutex

	links      chan string
	data       map[string]*GuideData
	sortedData []*GuideData
}

type Sorting struct {
	Field    string
	Strategy string
}

type ScraperParams struct {
	Directory string
	Formats   []string
	Sortings  []*Sorting
}

func NewScraper(p *ScraperParams) (*Scraper, error) {
	if p == nil || p.Formats == nil || len(p.Formats) == 0 || p.Sortings == nil {
		return nil, fmt.Errorf("failed to initialize scrapper, invalid parameters")
	}

	s := &Scraper{
		directory: p.Directory,
		formats:   p.Formats,
		sortings:  p.Sortings,

		lastPage: 1,

		wg:    sync.WaitGroup{},
		pmu:   sync.Mutex{},
		mu:    sync.Mutex{},
		links: make(chan string),
		data:  make(map[string]*GuideData),
	}

	collector := colly.NewCollector(
		colly.Async(true),
		colly.AllowURLRevisit(),
	)
	collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 6,
		Delay:       4 * time.Second,
	})
	collector.OnError(func(r *colly.Response, err error) {
		fmt.Printf("failed on link %s, reenqueueing link: %s\n", r.Request.URL.String(), err.Error())
		s.links <- r.Request.URL.String()
	})
	collector.OnResponse(func(r *colly.Response) {
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

	s.collector = collector
	return s, nil
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
			fmt.Printf("enqueueing %s\n", link)
			go s.collector.Visit(link)
		}
	}()

	s.wg.Wait()
	close(s.links)

	s.sortData()
	s.dumpData()
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
	if maxLastPage > s.lastPage {
		fmt.Printf("found new max page: %d\n", maxLastPage)
		for i := s.lastPage + 1; i <= maxLastPage; i++ {
			s.wg.Add(1)
			s.links <- fmt.Sprintf("%s%d", guidesPage, i)
		}
		s.lastPage = maxLastPage
	}
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

	platformList := []string{}
	doc.Find(".platforms").Children().Each(func(index int, item *goquery.Selection) {
		platformList = append(platformList, item.Text())
	})
	platforms := strings.Join(platformList, " ")

	overviewInfo := doc.Find(".overview-info")
	difficulty := overviewInfo.Find("span:nth-of-type(1)").Find("span:nth-of-type(1)").Text()
	timeNeeded := overviewInfo.Find("span:nth-of-type(3)").Find("span:nth-of-type(1)").Text()

	platinumInfo := doc.Find("img[alt='Platinum']").ParentsFiltered("tr").First()
	platinumRarity := platinumInfo.Children().Eq(platinumInfo.Children().Length() - 2).First().Children().First().Find("span").First().Text()

	s.mu.Lock()
	s.data[link] = &GuideData{
		Game:             game,
		Link:             link,
		Platforms:        platforms,
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

func (s *Scraper) sortData() {
	fmt.Print("sorting data\n")
	s.sortedData = make([]*GuideData, len(s.data))
	idx := 0
	for _, entry := range s.data {
		s.sortedData[idx] = entry
		idx++
	}

	if s.sortings == nil || len(s.sortings) == 0 {
		return
	}

	getDifficulty := func(i string) int {
		if i == "" {
			return -1
		}
		parts := strings.Split(i, "/")
		if len(parts) == 2 {
			r, err := strconv.Atoi(parts[0])
			if err != nil {
				return -1
			}
			return r
		}
		return -1
	}

	getInt := func(i string) int {
		if i == "" {
			return -1
		}
		r, err := strconv.Atoi(i)
		if err != nil {
			return -1
		}
		return r
	}

	getPlatinumRarity := func(i string) float64 {
		if i == "" {
			return -1
		}
		parts := strings.Split(i, "%")
		if len(parts) == 2 {
			r, err := strconv.ParseFloat(parts[0], 64)
			if err != nil {
				return -1
			}
			return r
		}
		return -1
	}

	compareI := func(i int, j int, strategy string) bool {
		if i < 0 {
			return strategy == SORT_STRATEGY_DESC
		}
		if j < 0 {
			return strategy == SORT_STRATEGY_ASC
		}
		if strategy == SORT_STRATEGY_ASC {
			return i < j
		}
		return i > j
	}

	compareF := func(i float64, j float64, strategy string) bool {
		if i < 0 {
			return strategy == SORT_STRATEGY_DESC
		}
		if j < 0 {
			return strategy == SORT_STRATEGY_ASC
		}
		if strategy == SORT_STRATEGY_ASC {
			return i < j
		}
		return i > j
	}

	sort.Slice(s.sortedData, func(i, j int) bool {
		sortedDataI := s.sortedData[i]
		sortedDataJ := s.sortedData[j]
		for _, sorting := range s.sortings {
			switch sorting.Field {
			case FIELD_DIFFUCULTY:
				{
					difficultyI := getDifficulty(sortedDataI.Difficulty)
					difficultyJ := getDifficulty(sortedDataJ.Difficulty)
					if difficultyI != difficultyJ {
						return compareI(difficultyI, difficultyJ, sorting.Strategy)
					}
				}
			case FIELD_TIME_NEEDED:
				{
					timeNeededI := getInt(sortedDataI.TimeNeeded)
					timeNeededJ := getInt(sortedDataJ.TimeNeeded)
					if timeNeededI != timeNeededJ {
						return compareI(timeNeededI, timeNeededJ, sorting.Strategy)
					}
				}
			case FIELD_PLATINUM_RARITY:
				{
					rarityI := getPlatinumRarity(sortedDataI.PlatinumRarity)
					rarityJ := getPlatinumRarity(sortedDataJ.PlatinumRarity)
					if rarityI != rarityJ && math.Abs(rarityI-rarityJ) > 0.005 {
						return compareF(rarityI, rarityJ, sorting.Strategy)
					}
				}
			case FIELD_VIEWS:
				{
					viewsI := getInt(sortedDataI.Views)
					viewsJ := getInt(sortedDataJ.Views)
					if viewsI != viewsJ {
						return compareI(viewsI, viewsJ, sorting.Strategy)
					}
				}
			case FIELD_GUIDE_RATING:
				{
					guideRatingI := getInt(sortedDataI.GuideRating)
					guideRatingJ := getInt(sortedDataJ.GuideRating)
					if guideRatingI != guideRatingJ {
						return compareI(guideRatingI, guideRatingJ, sorting.Strategy)
					}
				}
			case FIELD_GUIDE_RATING_COUNT:
				{
					guideRatingCountI := getInt(sortedDataI.GuideRatingCount)
					guideRatingCountJ := getInt(sortedDataJ.GuideRatingCount)
					if guideRatingCountI != guideRatingCountJ {
						return compareI(guideRatingCountI, guideRatingCountJ, sorting.Strategy)
					}
				}
			case FIELD_USER_FAVOURITES:
				{
					userFavouritesI := getInt(sortedDataI.UserFavourites)
					userFavouritesJ := getInt(sortedDataJ.UserFavourites)
					if userFavouritesI != userFavouritesJ {
						return compareI(userFavouritesI, userFavouritesJ, sorting.Strategy)
					}
				}
			default:
				{
				}
			}
		}
		return false
	})
}

func (s *Scraper) dumpData() {
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

func (s *Scraper) dumpJson() error {
	fmt.Print("exporting data to .json\n")
	file, err := os.Create(filepath.Join(s.directory, "guide_data.json"))
	if err != nil {
		return err
	}
	defer file.Close()

	jsonData, err := json.MarshalIndent(s.sortedData, "", " ")
	if err != nil {
		return err
	}

	if _, err := file.Write(jsonData); err != nil {
		return err
	}

	return nil
}

func (s *Scraper) dumpCsv() error {
	fmt.Print("exporting data to .csv\n")
	file, err := os.Create(filepath.Join(s.directory, "guide_data.csv"))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	entries := [][]string{
		{
			FIELD_GAME,
			FIELD_LINK,
			FIELD_PLATFORMS,
			FIELD_DIFFUCULTY,
			FIELD_TIME_NEEDED,
			FIELD_PLATINUM_RARITY,
			FIELD_VIEWS,
			FIELD_GUIDE_RATING,
			FIELD_GUIDE_RATING_COUNT,
			FIELD_USER_FAVOURITES,
		},
	}
	for _, entry := range s.sortedData {
		entries = append(entries, []string{
			entry.Game,
			entry.Link,
			entry.Platforms,
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
	fmt.Print("exporting data to .md\n")
	file, err := os.Create(filepath.Join(s.directory, "guide_data.md"))
	if err != nil {
		return err
	}
	defer file.Close()

	builder := strings.Builder{}
	builder.WriteString(
		fmt.Sprintf(
			"| **%s** | **%s** | **%s** | **%s** | **%s** | **%s** | **%s** | **%s** | **%s** |\n",
			FIELD_GAME,
			FIELD_PLATFORMS,
			FIELD_DIFFUCULTY,
			FIELD_TIME_NEEDED,
			FIELD_PLATINUM_RARITY,
			FIELD_VIEWS,
			FIELD_GUIDE_RATING,
			FIELD_GUIDE_RATING_COUNT,
			FIELD_USER_FAVOURITES,
		),
	)
	builder.WriteString("|:--------|:------:|:----:|:----:|:----:|:-----:|:----:|:-----:|:-----:|\n")

	for _, entry := range s.sortedData {
		game := entry.Game
		if game == "" {
			game = "Game name not found"
		}

		_, err = builder.WriteString(
			fmt.Sprintf(
				"| [%s](%s) | %s | %s | %s | %s | %s | %s | %s | %s |\n",
				game,
				entry.Link,
				entry.Platforms,
				entry.Difficulty,
				entry.TimeNeeded,
				entry.PlatinumRarity,
				entry.Views,
				entry.GuideRating,
				entry.GuideRatingCount,
				entry.UserFavourites,
			),
		)
		if err != nil {
			return err
		}
	}

	if _, err = file.WriteString(builder.String()); err != nil {
		return err
	}

	return nil
}
