package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

const (
	FORMAT_JSON = "json"
	FORMAT_CSV  = "csv"
	FORMAT_MD   = "md"
)

const (
	FIELD_GAME               = "game"
	FIELD_LINK               = "link"
	FIELD_PLATFORMS          = "platforms"
	FIELD_DIFFUCULTY         = "difficulty"
	FIELD_TIME_NEEDED        = "time_needed"
	FIELD_PLATINUM_RARITY    = "platinum_rarity"
	FIELD_VIEWS              = "views"
	FIELD_GUIDE_RATING       = "guide_rating"
	FIELD_GUIDE_RATING_COUNT = "guide_rating_count"
	FIELD_USER_FAVOURITES    = "user_favourites"
)

const (
	SORT_STRATEGY_ASC  = "asc"
	SORT_STRATEGY_DESC = "desc"
)

var formats = []string{
	FORMAT_JSON,
	FORMAT_CSV,
	FORMAT_MD,
}

var sortableFields = []string{
	FIELD_DIFFUCULTY,
	FIELD_TIME_NEEDED,
	FIELD_PLATINUM_RARITY,
	FIELD_VIEWS,
	FIELD_GUIDE_RATING,
	FIELD_GUIDE_RATING_COUNT,
	FIELD_USER_FAVOURITES,
}

func main() {
	app := &cli.App{
		Name:  "psnprofiles-guide-scraper",
		Usage: "A simple CLI application",
		Commands: []*cli.Command{
			{
				Name: "scrape",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:     "format",
						Aliases:  []string{"f"},
						Usage:    fmt.Sprintf("formats to export to, possible values %v", formats),
						Required: false,
					},
					&cli.StringFlag{
						Name:     "output-dir",
						Aliases:  []string{"o"},
						Usage:    "directory where data will be exported to",
						Required: false,
					},
					&cli.StringSliceFlag{
						Name:    "sort",
						Aliases: []string{"s"},
						Usage:   fmt.Sprintf("fields to sort by, appended by strategy (asc, desc), default is asc. example (platinum_rarity;asc), sort priority is based on order in which fields are given, possible values %v", sortableFields),
					},
				},
				Action: func(c *cli.Context) error {
					requestedFormats := c.StringSlice("format")
					for _, requestedFormat := range requestedFormats {
						formatAllowed := false
						for _, format := range formats {
							if requestedFormat == format {
								formatAllowed = true
							}
						}
						if !formatAllowed {
							return fmt.Errorf("unknown format '%s'", requestedFormat)
						}
					}

					requestedDirectory := c.String("output-dir")
					if requestedDirectory != "" {
						info, err := os.Stat(requestedDirectory)
						if err != nil {
							return fmt.Errorf("failed when checking whether %s is a valid directory or not: %s", requestedDirectory, err.Error())
						}
						if !info.IsDir() {
							return fmt.Errorf("%s is not a directory", requestedDirectory)
						}
					}

					requestedSortings := c.StringSlice("sort")
					sortings := []*Sorting{}
					for _, requestedSorting := range requestedSortings {
						parts := strings.Split(requestedSorting, ";")
						field := ""
						strategy := SORT_STRATEGY_ASC

						if len(parts) == 1 {
							field = parts[0]
						} else if len(parts) == 2 {
							field = parts[0]
							strategy = parts[1]
						} else {
							return fmt.Errorf("failed to parse requested sorting %s", requestedSorting)
						}

						sortingAllowed := false
						for _, sortableField := range sortableFields {
							if field == sortableField {
								sortingAllowed = true
							}
						}
						if !sortingAllowed {
							return fmt.Errorf("unknown sorting field '%s'", field)
						}

						if strategy != SORT_STRATEGY_ASC && strategy != SORT_STRATEGY_DESC {
							return fmt.Errorf("unknown sorting strategy '%s'", strategy)
						}

						sortings = append(sortings, &Sorting{
							Field:    field,
							Strategy: strategy,
						})
					}

					scraper, err := NewScraper(&ScraperParams{
						Directory: requestedDirectory,
						Formats:   requestedFormats,
						Sortings:  sortings,
					})
					if err != nil {
						return err
					}
					scraper.init()
					scraper.scrape()
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println("failed to scrape psnprofiles,", err)
		os.Exit(1)
	}
}
