#!/bin/bash
nodeip=$1
iface=$2
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--node-ip=${nodeip} --flannel-iface=${iface}" sh -
