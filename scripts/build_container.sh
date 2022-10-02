#!/bin/bash

echo "building image for $1"
sudo docker build -t $1 ~/mesh/mesh-bw-scheduler/containers/$1/.
sudo docker save --output ~/mesh/images/$1.tar $1:latest
