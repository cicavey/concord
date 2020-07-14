default:
	go build -ldflags="-s -w" -o concord cmd/*.go

# For Pi-Zero
arm6:
	GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w" -o concord.arm6 cmd/*.go

# For other Pi
arm7:
	GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o concord.arm7 cmd/*.go

dist: default arm6 arm7
	upx --brute concord*

clean:
	rm -f concord concord.*