localName = ali.out
linuxName = ali_linux.out
macName = ali_mac.out

local: clean
	go build -ldflags '-s -w' -o $(localName) *.go
	mkdir -p tmp/server
	mkdir -p tmp/client
	cp $(localName) tmp/server
	cp $(localName) tmp/client
	cp config.json tmp/server
	cp config.json tmp/client
linux: clean
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o $(linuxName) *.go
	upx --best $(linuxName)
mac: clean
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags '-s -w' -o $(macName) *.go
	upx --best $(macName)
clean:
	find . -name "*.log" | xargs rm -f
	find . -name "*.out" | xargs rm -f
