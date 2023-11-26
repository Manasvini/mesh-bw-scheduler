## bw_shaping  
This folder contains code to do traffic shaping on multiple links.  
To setup, make sure you have a cloudlab topology set up (with VMs). Refer to this [topology](https://www.cloudlab.us/status.php?uuid=9adf3477-736f-11ee-9f39-e4434b2381fc) for an example. It sets up a mesh of routers (node0 to node4) and connects compute nodes (node5 to node9) to them.  
Copy the hostnames and ensure that the nodename parameter in your config file is correct. Look at `config_5nodes.json` for reference.  
Each node has its own set of interfaces on which we run `tc`. The node specific configs are specified in `5nodes/nodeX.json`. The bandwidth values to shape for each interface are specified in this file. To set up all the nodes, do:  
```shell  
$ python3 set_bw.py --config config_5nodes.json --user your_cloudlab_username  
```
Once you have verified that `scp` succeeded on all the nodes and the config files and bw files have been copied over, then you ssh into each node and set up traffic shaping on each node like so:  
```shell  
$ python3 bw_controller.py nodeX.json  
```
