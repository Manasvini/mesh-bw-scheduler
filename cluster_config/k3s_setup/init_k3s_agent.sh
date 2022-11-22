#!/bin/bash

server_ip=$1
token=$2

curl -sfL https://get.k3s.io | K3S_URL=https://$server_ip:6443 K3S_TOKEN=$token sh -  &> $(hostname)_agent.log &
