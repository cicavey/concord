default:
	go build -o concord cmd/main.go

# For Pi-Zero
arm6:
	GOOS=linux GOARCH=arm GOARM=6 go build -o concord.arm6 cmd/main.go

# For other Pi
arm7:
	GOOS=linux GOARCH=arm GOARM=7 go build -o concord.arm7 cmd/main.go

clean:
	rm concord*