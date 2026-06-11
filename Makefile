.PHONY: build

build:
	GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -o printbridge.exe .
