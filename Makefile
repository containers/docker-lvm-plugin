# Installation Directories
SYSCONFDIR ?=$(DESTDIR)/etc/docker
SYSTEMDIR ?=$(DESTDIR)/usr/lib/systemd/system
GOLANG ?= /usr/bin/go
BINARY ?= docker-lvm-plugin
MANINSTALLDIR?= ${DESTDIR}/usr/share/man
BINDIR ?=$(DESTDIR)/usr/libexec/docker

export GO15VENDOREXPERIMENT=1

all: man lvm-plugin-build

.PHONY: man
man:
	go-md2man -in man/docker-lvm-plugin.8.md -out docker-lvm-plugin.8

.PHONY: lvm-plugin-build
lvm-plugin-build: main.go driver.go
	$(GOLANG) build -o $(BINARY) .

.PHONY: install
install:
	install -D -m 755 etc/docker/docker-lvm-plugin $(SYSCONFDIR)/docker-lvm-plugin
	install -D -m 644 systemd/docker-lvm-plugin.service $(SYSTEMDIR)/docker-lvm-plugin.service
	install -D -m 644 systemd/docker-lvm-plugin.socket $(SYSTEMDIR)/docker-lvm-plugin.socket
	install -D -m 755 $(BINARY) $(BINDIR)/$(BINARY)
	install -D -m 644 docker-lvm-plugin.8 ${MANINSTALLDIR}/man8/docker-lvm-plugin.8

.PHONY: clean
clean:
	rm -f $(BINARY)
	rm -f docker-lvm-plugin.8


