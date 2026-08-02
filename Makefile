NAME    = quadlet-compose
VERSION = 0.1.0
LDFLAGS = -ldflags "-X main.version=$(VERSION)"

.PHONY: build test install vendor dist clean completions

build:
	go build $(LDFLAGS) -o $(NAME) .
	ln -sf $(NAME) complet

test:
	go test -v -cover ./...

install: build
	install -Dpm 0755 $(NAME) /usr/local/bin/$(NAME)

vendor:
	go mod vendor

dist: vendor
	mkdir -p $(NAME)-$(VERSION)
	cp -a cmd internal main.go go.mod go.sum vendor testdata quadlet-compose.spec LICENSE Makefile $(NAME)-$(VERSION)/
	tar czf $(NAME)-$(VERSION).tar.gz $(NAME)-$(VERSION)
	rm -rf $(NAME)-$(VERSION)

clean:
	rm -f $(NAME)
	rm -rf $(NAME)-$(VERSION) $(NAME)-$(VERSION).tar.gz
	rm -rf vendor
	rm -rf completions

completions: build
	mkdir -p completions
	./$(NAME) completions bash > completions/$(NAME).bash
	./$(NAME) completions zsh > completions/_$(NAME)
	./$(NAME) completions fish > completions/$(NAME).fish
