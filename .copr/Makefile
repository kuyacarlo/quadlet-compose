NAME = quadlet-compose
VERSION = $(shell grep '^Version:' ../$(NAME).spec | awk '{print $$2}')
TARBALL = $(NAME)-$(VERSION).tar.gz
SRCDIR = $(shell cd .. && pwd)

srpm:
	cd $(SRCDIR) && \
	mkdir -p $(NAME)-$(VERSION) && \
	cp -a cmd internal main.go go.mod go.sum vendor testdata $(NAME).spec $(NAME).1 LICENSE Makefile .gitignore $(NAME)-$(VERSION)/ && \
	tar czf $(TARBALL) $(NAME)-$(VERSION) && \
	rm -rf $(NAME)-$(VERSION)

	rpmbuild -bs $(SRCDIR)/$(NAME).spec \
		--define "_sourcedir $(SRCDIR)" \
		--define "_srcrpmdir $(outdir)"
