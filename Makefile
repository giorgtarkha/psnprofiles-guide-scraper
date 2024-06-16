GO_CMD := go run main.go scraper.go

OUTPUT_DIR ?= ../

all:
	cd ./cmd && $(GO_CMD) scrape -f json -f csv -f md -o $(OUTPUT_DIR)

json:
	cd ./cmd && $(GO_CMD) scrape -f json -o $(OUTPUT_DIR)

csv:
	cd ./cmd && $(GO_CMD) scrape -f csv -o $(OUTPUT_DIR)

md:
	cd ./cmd && $(GO_CMD) scrape -f md -o $(OUTPUT_DIR)
