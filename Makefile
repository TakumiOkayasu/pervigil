.PHONY: build build-linux test clean deploy

BINARY_NAME := pervigil-bot
BUILD_DIR := bin
BOT_DIR := bot

build:
	cd $(BOT_DIR) && go build -o ../$(BUILD_DIR)/$(BINARY_NAME) ./cmd/pervigil-bot

build-linux:
	cd $(BOT_DIR) && GOOS=linux GOARCH=amd64 go build -o ../$(BUILD_DIR)/$(BINARY_NAME) ./cmd/pervigil-bot

test:
	cd $(BOT_DIR) && go test ./...

clean:
	rm -rf $(BUILD_DIR)

deps:
	cd $(BOT_DIR) && go mod download

tidy:
	cd $(BOT_DIR) && go mod tidy

# Deploy to VyOS (set VYOS_HOST)
deploy: build-linux
	scp $(BUILD_DIR)/$(BINARY_NAME) vyos@$(VYOS_HOST):/config/scripts/
	scp deploy/pervigil-bot.service vyos@$(VYOS_HOST):/tmp/
	ssh vyos@$(VYOS_HOST) 'sudo mv /tmp/pervigil-bot.service /etc/systemd/system/ && sudo systemctl daemon-reload && sudo systemctl enable pervigil-bot'
