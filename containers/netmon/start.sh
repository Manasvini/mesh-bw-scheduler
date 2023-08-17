#!/bin/bash
PROJ_ROOT=/users/msethur1/mesh-bw-scheduler/containers/netmon
## start iperf3
iperf3 -s > /dev/null  2>&1 &

## start net_helper  
cd $PROJ_ROOT/net_helper
python3 net_helper.py --config cloudlab_config.json  > /dev/null 2>&1 &  


## start netmon  
cd $PROJ_ROOT/netmon_main
sudo -E ./netmon_main -config config_cloudlab.txt -device eno1
