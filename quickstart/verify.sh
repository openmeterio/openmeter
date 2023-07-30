#!/bin/bash

temp_file=$(mktemp)

curl http://localhost:8888/api/v1/meters/m1/values?windowSize=HOUR >$temp_file

[ $(cat $temp_file | jq '.data[0].value') == 2 ] || (echo "Unexpected value" && cat $temp_file && exit 1)
[ $(cat $temp_file | jq '.data[1].value') == 1 ] || (echo "Unexpected value" && cat $temp_file && exit 1)
