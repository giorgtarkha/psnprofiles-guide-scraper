# PSN Profiles Guide Scraper

----

This tool is used to scrape [PSN Profiles Guides](https://psnprofiles.com/guides/popular). 
It is useful mainly for trophy hunters, to look for games based on difficulty, platinum rarity and etc.

> [!WARNING]
> This is a scraper, so naturally if website structure changes, it is possible for the script to break, don't consider the script to always be up to date.
> 
> PSN profiles guides structure is not consistent, so it's possible for some fields to have abnormal data.
> 
> PSN profiles has rate limiting from same source, best configuration I found was 6 parallel requests with 4 second delays, so it will take some time for all guides to be scraped, around 1 hour.  

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

> [!NOTE]
> Available output formats:<br>
> **[ json, csv, md ]**
>
> Available sortable fields:<br>
> **[ difficulty, time_needed, platinum_rarity, views, guide_rating, guide_rating_count, user_favourites ]**<br>
>
> Sorting priority is based on which sorting field is provided first. Sorting applied by default if Makefile is used (All in ascending order):<br>
> **[ difficulty, platinum_rarity, time_needed, views ]**<br>

### Running using Make

```shell
# runs the script and puts output in every format in same directory as the makefile
$ make

# runs the script and puts output in given format in same directory as the makefile.
$ make {format}

# runs the script and puts output in given format (or all formats) in given directory. 
# If relative path is given, it is relative to ./cmd
$ make OUTPUT_DIR={dir_to_export_to}
```

### Running using Go

```shell
# runs the script and puts output in requested formats, at least one format is required. 
# By default output is put in same directory as the scripts.
$ go run cmd/* scrape -f {format1} -f {format2} ...

# runs the script and puts output in requested directory.
$ go run cmd/* scrape -o {output_dir} ...

# runs the script and sorts scraped data based on fields. 
# Field priority is based on the order of input. By default ascending order is used. 
# Sort order can also be explicitly provided, for example: 
# -s difficulty;asc -s platinum_rarity;desc
$ go run cmd/* scrape -s {sorting1} -s {sorting2} ...
```
