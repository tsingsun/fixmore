version := $(shell /bin/date "+%Y-%m-%d %H:%M")
BUILD_NAME=fix

build:
	go build -ldflags="-s -w" -ldflags="-X 'main.BuildTime=$(version)'" -o ./cmd/tradeclient/$(BUILD_NAME) ./cmd/tradeclient/
	$(if $(shell command -v upx), upx $(BUILD_NAME))
mac:
	GOOS=darwin go build -ldflags="-s -w" -ldflags="-X 'main.BuildTime=$(version)'" -o ./cmd/tradeclient/$(BUILD_NAME)-darwin ./cmd/tradeclient/
	$(if $(shell command -v upx), upx $(BUILD_NAME)-darwin)
win:
	GOOS=windows go build -ldflags="-s -w" -ldflags="-X 'main.BuildTime=$(version)'" -o ./cmd/tradeclient/$(BUILD_NAME).exe ./cmd/tradeclient/
	$(if $(shell command -v upx), upx $(BUILD_NAME).exe)
linux:
	GOOS=linux go build -ldflags="-s -w" -ldflags="-X 'main.BuildTime=$(version)'" -o ./cmd/tradeclient/$(BUILD_NAME)-linux ./cmd/tradeclient/
	$(if $(shell command -v upx), upx $(BUILD_NAME)-linux)