NAME=go-basher
OWNER=progrium
BASH_DIR=.bash
BASH_STATIC_VERSION=5.0

test:
	go test -v

build:
	go install || true

deps:
	go get -u github.com/jteeuwen/go-bindata/...

bash:
	# Don't run if you don't have to. Adds several megs to repo with every commit.
	rm -rf $(BASH_DIR) && mkdir -p $(BASH_DIR)/linux $(BASH_DIR)/osx

	curl -#SLk https://github.com/robxu9/bash-static/releases/download/$(BASH_STATIC_VERSION)/bash-linux \
		> $(BASH_DIR)/linux/bash && strip $(BASH_DIR)/linux/bash

	curl -#SLk https://github.com/robxu9/bash-static/releases/download/$(BASH_STATIC_VERSION)/bash-osx \
		> $(BASH_DIR)/osx/bash

	chmod +x $(BASH_DIR)/*/bash

	# if upx is present, compress bash binaries
	if [ `command -v upx` ]; then upx --8mib-ram --best --brute --ultra-brute -9 $(BASH_DIR)/*/bash; fi

	go-bindata -tags=linux -o=bash_linux.go -prefix=$(BASH_DIR)/linux -pkg=basher $(BASH_DIR)/linux
	go-bindata -tags=darwin -o=bash_darwin.go -prefix=$(BASH_DIR)/osx -pkg=basher $(BASH_DIR)/osx

circleci:
	rm ~/.gitconfig
	rm -rf /home/ubuntu/.go_workspace/src/github.com/$(OWNER)/$(NAME) && cd .. \
		&& mkdir -p /home/ubuntu/.go_workspace/src/github.com/$(OWNER) \
		&& mv $(NAME) /home/ubuntu/.go_workspace/src/github.com/$(OWNER)/$(NAME) \
		&& ln -s /home/ubuntu/.go_workspace/src/github.com/$(OWNER)/$(NAME) $(NAME)
