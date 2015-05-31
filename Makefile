BINDIR=.bin

bindata:
	go-bindata -tags="linux" -o="./bindata_linux.go" -prefix="$(BINDIR)/linux" -pkg="basher" $(BINDIR)/linux
	go-bindata -tags="darwin" -o="./bindata_darwin.go" -prefix="$(BINDIR)/osx" -pkg="basher" $(BINDIR)/osx

deps:
	go get -u github.com/jteeuwen/go-bindata/...
	mkdir -p $(BINDIR)/linux $(BINDIR)/osx
	curl -Lo $(BINDIR)/linux/bash https://github.com/robxu9/bash-static/releases/download/4.3.30/bash-linux
	curl -Lo $(BINDIR)/osx/bash https://github.com/robxu9/bash-static/releases/download/4.3.30/bash-osx
	chmod +x $(BINDIR)/linux/* $(BINDIR)/osx/*

.PHONY: deps bindata
