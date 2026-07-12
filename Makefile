BINARY := bin/secmsg
PKG := ./cmd/secmsg

.PHONY: build clean

build:
	go build -o $(BINARY) $(PKG)

clean:
	rm -rf bin
