GO_CMD := go run main.go scraper.go

OUTPUT_DIR ?= ../

all:
	cd ./cmd && $(GO_CMD) scrape -f json -f csv -f md -s difficulty -s platinum_rarity -s time_needed -s views -o $(OUTPUT_DIR)

json:
	cd ./cmd && $(GO_CMD) scrape -f json -s platinum_rarity -o $(OUTPUT_DIR)

csv:
	cd ./cmd && $(GO_CMD) scrape -f csv -s platinum_rarity -o $(OUTPUT_DIR)

md:
	cd ./cmd && $(GO_CMD) scrape -f md -s platinum_rarity -o $(OUTPUT_DIR)
