#!/usr/bin/env bash
set -euo pipefail

if [ ! -f config.yaml ]; then
  cp config.example.yaml config.yaml
fi

if [ config.yaml -ot config.example.yaml ]; then
  diff -u config.yaml config.example.yaml || {
    echo "!!! The configuration example changed. Please update your config.yaml file accordingly (or at least touch it). !!!"
    exit 1
  }
fi
