BUILD_VERSION := $(shell git describe --always)
BUILD_DATE := $(shell date +'%Y-%m-%d_%T')
default:
	go build -ldflags="-s -w -X main.BuildVersion=$(BUILD_VERSION) -X main.BuildDate=$(BUILD_DATE)" -o concord cmd/*.go

# For Pi-Zero
arm6:
	GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w -X main.BuildVersion=$(BUILD_VERSION) -X main.BuildDate=$(BUILD_DATE)" -o concord.arm6 cmd/*.go

# For other Pi
arm7:
	GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w -X main.BuildVersion=$(BUILD_VERSION) -X main.BuildDate=$(BUILD_DATE)" -o concord.arm7 cmd/*.go

dist: default arm6 arm7
	upx --brute concord*

clean:
	rm -f concord concord.*