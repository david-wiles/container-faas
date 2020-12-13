#!/bin/sh

walk_dir() {
	for pathname in "$1"/*; do
		if [ -d "$pathname" ]; then
		    walk_dir "$pathname"
        elif [ -e "$pathname" ]; then
            CTX=$(dirname "$pathname")
#            printf "docker build %s -t %s\n" "$CTX" $(basename $(dirname "$pathname"))
            docker build "$CTX" -t $(basename "$CTX")
        fi
    done
}

# build all docker images in the assets/dockerfiles/ directory
walk_dir "../assets/dockerfiles"

# Create a network
docker network create app-network
