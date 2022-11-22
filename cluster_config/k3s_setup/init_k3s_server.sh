#!/bin/bash

username=$(whoami)
group=$(id -gn)
curl -sfL https://get.k3s.io | sh - &>  k3s_server.log &
sudo cat /var/lib/rancher/k3s/server/node-token >> ./token.txt
sudo chown $username:$group token.txt
