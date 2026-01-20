#!/usr/bin/env sh

docker run -it --rm -p 5173:5173 -v "$(pwd):/src" -w "/src/webui" node:20 /bin/bash