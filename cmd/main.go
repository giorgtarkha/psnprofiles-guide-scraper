package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

const (
	FORMAT_JSON = "json"
	FORMAT_CSV  = "csv"
	FORMAT_MD   = "md"
)

var formats = []string{
	FORMAT_JSON,
	FORMAT_CSV,
	FORMAT_MD,
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
						Usage:    fmt.Sprintf("list of formats to export to, possible values %v", formats),
						Required: false,
					},
					&cli.StringFlag{
						Name:     "output-dir",
						Aliases:  []string{"o"},
						Usage:    "directory where data will be exported to",
						Required: false,
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

					scraper := NewScraper(requestedDirectory, requestedFormats)
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
