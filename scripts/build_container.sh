#!/bin/bash

echo "building image for $1"
sudo docker build -t $1 ~/mesh/tmp/$1/.
sudo docker save --output ~/mesh/tmp/$1.tar $1:latest
