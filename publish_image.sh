#!/bin/bash

if [ -z ${1+x} ];
then
    echo "Please specify a tag to build the image"
    exit 1
fi

image_name="ghcr.io/uocsclub/aoc-lb:$1"

docker build -t $image_name . 

echo -e "Built container $image_name"

docker push $image_name
