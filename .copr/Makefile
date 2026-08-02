NAME = quadlet-compose
VERSION = 0.1.0
TARBALL = $(NAME)-$(VERSION).tar.gz
SRCDIR = $(shell cd .. && pwd)

srpm:
	# Vendor dependencies (requires networking enabled in COPR)
	cd $(SRCDIR) && go mod vendor

	# Create source tarball
	cd $(SRCDIR) && \
	mkdir -p $(NAME)-$(VERSION) && \
	cp -a cmd internal main.go go.mod go.sum vendor testdata quadlet-compose.spec LICENSE Makefile .gitignore $(NAME)-$(VERSION)/ && \
	tar czf $(TARBALL) $(NAME)-$(VERSION) && \
	rm -rf $(NAME)-$(VERSION)

	# Build SRPM
	rpmbuild -bs $(SRCDIR)/quadlet-compose.spec \
		--define "_sourcedir $(SRCDIR)" \
		--define "_srcrpmdir $(outdir)"
