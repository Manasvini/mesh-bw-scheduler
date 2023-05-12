#!/bin/bash

token=K10e995907a9fa1a0c5eb17aa2e6a6773ad12a9708eddc0ac19156bf2a86d8c43b9::server:ba447d5c88f09384f53814058e33896f
curl -sfL https://get.k3s.io | K3S_URL=https://172.17.2.6:6443 K3S_TOKEN=$token