#!/bin/sh

walk_dir() {
	for pathname in "$1"/*; do
	    FILE=$(basename "$pathname")
		if [ -d "$pathname" ]; then
		    walk_dir "$pathname"
        elif [ "$FILE" = "Dockerfile" ]; then
            CTX=$(dirname "$pathname")
#            printf "docker build %s -t %s\n" "$CTX" $(basename $(dirname "$pathname"))
            docker build -t $(basename "$CTX") "$CTX"
        fi
    done
}

# build all docker images in the assets/dockerfiles/ directory
walk_dir "../assets/dockerfiles"

# Create a network
docker network create app-network
