.PHONY: all
all: network images build

network:
	docker network create app-network

images:
	./tools/build_images.sh

build:
	docker build -t paas-server .
