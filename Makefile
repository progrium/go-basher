NAME=go-basher
OWNER=progrium
BASH_DIR=.bash
BASH_STATIC_VERSION=5.1.008-1.2.2

test:
	go test -v

build:
	go install

deps:
	go get -u github.com/a-urth/go-bindata/...

bash:
	# Don't run if you don't have to. Adds several megs to repo with every commit.
	rm -rf $(BASH_DIR) && mkdir -p $(BASH_DIR)/linux-arm $(BASH_DIR)/linux-arm64 $(BASH_DIR)/linux-amd64 $(BASH_DIR)/osx-arm64 $(BASH_DIR)/osx-amd64

	curl -#SLk https://github.com/robxu9/bash-static/releases/download/$(BASH_STATIC_VERSION)/bash-linux-aarch64 \
		> $(BASH_DIR)/linux-arm64/bash

	curl -#SLk https://github.com/robxu9/bash-static/releases/download/$(BASH_STATIC_VERSION)/bash-linux-armv7 \
		> $(BASH_DIR)/linux-arm/bash

	curl -#SLk https://github.com/robxu9/bash-static/releases/download/$(BASH_STATIC_VERSION)/bash-linux-x86_64 \
		> $(BASH_DIR)/linux-amd64/bash

	curl -#SLk https://github.com/robxu9/bash-static/releases/download/$(BASH_STATIC_VERSION)/bash-macos-aarch64 \
		> $(BASH_DIR)/osx-arm64/bash

	curl -#SLk https://github.com/robxu9/bash-static/releases/download/$(BASH_STATIC_VERSION)/bash-macos-x86_64 \
		> $(BASH_DIR)/osx-amd64/bash

	chmod +x $(BASH_DIR)/*/bash

	go-bindata -tags=linux,arm -o=bash_linux_arm.go -prefix=$(BASH_DIR)/linux-arm -pkg=basher $(BASH_DIR)/linux-arm
	go-bindata -tags=linux,arm64 -o=bash_linux_arm64.go -prefix=$(BASH_DIR)/linux-arm64 -pkg=basher $(BASH_DIR)/linux-arm64
	go-bindata -tags=linux,amd64 -o=bash_linux_amd64.go -prefix=$(BASH_DIR)/linux-amd64 -pkg=basher $(BASH_DIR)/linux-amd64
	go-bindata -tags=darwin,arm64 -o=bash_darwin_arm64.go -prefix=$(BASH_DIR)/osx-arm64 -pkg=basher $(BASH_DIR)/osx-arm64
	go-bindata -tags=darwin,amd64 -o=bash_darwin_amd64.go -prefix=$(BASH_DIR)/osx-amd64 -pkg=basher $(BASH_DIR)/osx-amd64
