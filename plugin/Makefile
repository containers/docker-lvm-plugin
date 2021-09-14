name := containers/docker-lvm-plugin

.PHONY: build
build: docker-lvm-plugin

docker-lvm-plugin: main.go driver.go utils.go go.mod go.sum
	docker run --rm --tmpfs /tmp -v $(shell pwd):/tmp/docker-lvm-plugin -w /tmp/docker-lvm-plugin golang:latest go build

.PHONY: create
create: plugin/config.json plugin/rootfs
	docker plugin rm --force $(name) || true
	docker plugin create $(name) plugin

.PHONY: install
install: create
	docker plugin rm --force $(name) || true
	docker plugin install $(name) --grant-all-permissions

.PHONY: push
push: create
	docker plugin push $(name)

.PHONY: enable
enable: create
	docker plugin enable $(name)

.PHONY: clean
clean:
	rm -rf plugin

plugin/rootfs: .dockerignore plugin/Dockerfile docker-lvm-plugin
	docker build --tag $(name):rootfs -f plugin/Dockerfile .
	docker rm --force --volumes rootfs || true
	docker create --name rootfs $(name):rootfs
	rm -rf $@
	mkdir -p $@
	docker export rootfs | tar -x -C $@
	docker rm --force --volumes rootfs

