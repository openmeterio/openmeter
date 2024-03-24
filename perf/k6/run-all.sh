#!/usr/bin/env sh

set -e

# check that k6 is installed
if ! [ -x "$(command -v k6)" ]; then
    echo 'Error: k6 is not installed.' >&2
    exit 1
fi

# list all files ending with .js in dist folder
for file in $(find dist -name "*.js"); do
    k6 run $file
done
