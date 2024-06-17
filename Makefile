GO_CMD := go run main.go scraper.go
SORTING := -s difficulty -s platinum_rarity -s time_needed -s views
OUTPUT_DIR ?= ../

all:
	cd ./cmd && $(GO_CMD) scrape -f json -f csv -f md $(SORTING) -o $(OUTPUT_DIR)

json:
	cd ./cmd && $(GO_CMD) scrape -f json $(SORTING) -o $(OUTPUT_DIR)

csv:
	cd ./cmd && $(GO_CMD) scrape -f csv $(SORTING) -o $(OUTPUT_DIR)

md:
	cd ./cmd && $(GO_CMD) scrape -f md $(SORTING) -o $(OUTPUT_DIR)
