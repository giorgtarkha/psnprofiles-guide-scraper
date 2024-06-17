# PSN Profiles Guide Scraper

----

This tool is used to scrape [PSN Profiles Guides](https://psnprofiles.com/guides/popular). 
It is useful mainly for trophy hunters, to look for games based on difficulty, platinum rarity and etc.

> [!WARNING]
> This is a scraper, so naturally if website structure changes, it is possible for the script to break, don't consider the script to always be up to date.
> 
> PSN profiles guides structure is not consistent, so it's possible for some fields to have abnormal data.
> 
> PSN profiles has rate limiting from same source, best configuration I found was 6 parallel requests with 4 second delays, so it will take some time for the whole website to be scraped, around 1 hour.  

## Fetched data
> **Keep in mind that github fails to display some formats so other means to view the data has to be used.**<br/>
> **Markdown gets displayed, but some rows get skipped**<br/>
> **CSV gets displayed without prettifying it**<br/>
> 
> * [Guide Data in CSV Format](https://github.com/giorgtarkha/psnprofiles-guide-scraper/blob/main/data/guide_data.csv)  
> * [Guide Data in JSON Format](https://github.com/giorgtarkha/psnprofiles-guide-scraper/blob/main/data/guide_data.json)  
> * [Guide Data in Markdown Format](https://github.com/giorgtarkha/psnprofiles-guide-scraper/blob/main/data/guide_data.md)

## Usage

This is used mostly as a one-time script, so no fancy stuff, no building, just running onces and that's it.
```
Available sortable fields:  'difficulty', 'time_needed', 'platinum_rarity', 'views', 'guide_rating', 'guide_rating_count', 'user_favourites'
Available output formats: 'json', 'csv', 'md'
Sorting priority is based on which sorting field is provided first.
```

### Running using Make

```
Sorting applied by default if Makefile is used is: 'difficulty', 'platinum_rarity', 'time_needed', 'views' (All in ascending order)

> make
runs the script and puts output in every format in same directory as the makefile.

> make json
runs the script and puts output in json format in same directory as the makefile.

> make csv
runs the script and puts output in csv format in same directory as the makefile.

> make md
runs the script and puts output in md format in same directory as the makefile.

> make $(format)? OUTPUT_DIR=$(dir_to_export_to)
runs the script and puts output in given format (or all formats) in given directory, if relative path is given, it is relative to $(makefile_dir)/cmd.
```

### Running using Go

```
cd cmd

> go run main.go scraper.go scrape -f $(format1) -f $(format2) ....
runs the script and puts output in requested formats, at least one format is required. By default output is put in same directory as the scripts.

> go run main.go scraper.go scrape -o $(output_dir) ....
runs the script and puts output in requested directory.

> go run main.go scraper.go scrape -s $(sorting1) -s $(sorting2) ....
runs the script and sorts scraped data based on fields, field priority is taken from the order of input.
By default ascending order is used, sort order can also be explicitly provided, for example: -s difficulty;asc -s platinum_rarity;desc
```
