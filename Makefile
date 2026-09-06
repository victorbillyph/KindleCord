APP=kindlecord
PKG=kindlecord
BUILD_DIR=build
GOFLAGS=-trimpath

.PHONY: all build build-arm build-all clean install test run sim strip package

all: build

build:
	go build $(GOFLAGS) -ldflags="-s -w" -o $(BUILD_DIR)/$(APP) ./cmd/kindlecord
	@echo "Built $(BUILD_DIR)/$(APP) ($$(du -h $(BUILD_DIR)/$(APP) | cut -f1))"

build-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build $(GOFLAGS) -ldflags="-s -w" -o $(BUILD_DIR)/$(APP)-arm ./cmd/kindlecord
	@echo "Built ARM $(BUILD_DIR)/$(APP)-arm ($$(du -h $(BUILD_DIR)/$(APP)-arm | cut -f1))"

build-all: build build-arm

clean:
	rm -rf $(BUILD_DIR)/*.pyc kindlecord/__pycache__

install: build-arm
	@echo "Copy to Kindle: scp $(BUILD_DIR)/$(APP)-arm root@kindle:/mnt/us/extensions/KindleCord/kindlecord"

test:
	go vet ./...
	go test ./... 2>&1 | head -n 50

run: build
	./$(BUILD_DIR)/$(APP)

sim: build
	mkdir -p data
	./$(BUILD_DIR)/$(APP)

strip:
	strip $(BUILD_DIR)/$(APP)* 2>/dev/null || true
	ls -lh $(BUILD_DIR)/

# Build the KUAL extension zip (drop the inner KindleCord/ folder into
# /mnt/us/extensions to install). Produces $(BUILD_DIR)/KindleCord.zip
package: build-arm
	rm -rf /tmp/kc-staging/KindleCord
	mkdir -p /tmp/kc-staging/KindleCord
	cp $(BUILD_DIR)/$(APP)-arm /tmp/kc-staging/KindleCord/kindlecord
	cp -r bin menu.json config.xml README.md /tmp/kc-staging/KindleCord/
	mkdir -p data
	cp data/config.example.json /tmp/kc-staging/KindleCord/data.example.json
	cd /tmp/kc-staging && rm -f $(CURDIR)/$(BUILD_DIR)/KindleCord.zip && zip -r $(CURDIR)/$(BUILD_DIR)/KindleCord.zip KindleCord/
	ls -lh $(BUILD_DIR)/KindleCord.zip
	@echo "Unzip to /mnt/us/extensions to install."
