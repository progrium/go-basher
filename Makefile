NAME=go-basher
OWNER=progrium
BASH_DIR=.bash
VERSION=v2

test:
	go test -v .

build:
	go install || true

deps:
	go get -u github.com/jteeuwen/go-bindata/...
	go get -u github.com/progrium/gh-release/...

bash:
	rm -rf $(BASH_DIR) && mkdir -p $(BASH_DIR)/linux $(BASH_DIR)/osx
	curl -Ls https://github.com/robxu9/bash-static/releases/download/4.3.30/bash-linux \
		> $(BASH_DIR)/linux/bash
	curl -Ls https://github.com/robxu9/bash-static/releases/download/4.3.30/bash-osx \
		> $(BASH_DIR)/osx/bash
	chmod +x $(BASH_DIR)/**/*
	go-bindata -tags=linux -o=bash_linux.go -prefix=$(BASH_DIR)/linux -pkg=basher $(BASH_DIR)/linux
	go-bindata -tags=darwin -o=bash_darwin.go -prefix=$(BASH_DIR)/osx -pkg=basher $(BASH_DIR)/osx

circleci:
	rm ~/.gitconfig
	rm -rf /home/ubuntu/.go_workspace/src/github.com/$(OWNER)/$(NAME) && cd .. \
		&& mkdir -p /home/ubuntu/.go_workspace/src/github.com/$(OWNER) \
		&& mv $(NAME) /home/ubuntu/.go_workspace/src/github.com/$(OWNER)/$(NAME) \
		&& ln -s /home/ubuntu/.go_workspace/src/github.com/$(OWNER)/$(NAME) $(NAME)

release:
	rm -rf release && mkdir release
	mv go-workspace.tgz release/$(NAME)_$(VERSION)_workspace.tgz
	gh-release create $(OWNER)/$(NAME) $(VERSION) $(shell git rev-parse --abbrev-ref HEAD) $(VERSION)
