## netmon tools  
This directory contains code for monitoring network statistics. There are two parts to the network monitoring: capacity (available bw) and usage (actual traffic).   
### Capacity  
The `net_helper` folder contains a python flask server that runs iperf and traceroutes for  a specified set of hosts. The set of hosts is specified in `net_helper/config.json`. 
#### Install deps  
```shell  
$ sudo apt install traceroute  
$ pip3 install -r requirements.txt  
```
#### Running 
```shell  
$ python3 net_helper.py  
```   
### Traffic monitoring and exposing metrics     
We use a BPF program to monitor the source/destination of packets and the size of each packet. The BPF program is in C (`packet.h`) and has a Go wrapper around it. The Go program also instantiates a gRPC server to expose the bandwidth and traffic metrics.  
#### Setup  
The setup is a little wonky because bcc (BPF Compiler Collection) from IOVisor is broken for Ubuntu 20.04. We need to first install BCC from source.   
Setup the dependencies first  
```shell  
$ sudo apt install -y bison build-essential cmake flex git libedit-dev \
  llvm-10 llvm-10-dev clang-10 libclang-10-dev python zlib1g-dev libelf-dev libfl-dev python3-setuptools  
```   
Clone the bcc repo:  
```shell  
$ git clone --recurse-submodules  https://github.com/iovisor/bcc.git  
```  
Replace `bcc/cmake/clang_libs.cmake` with the one in the `install/` directory of this repo.   
Build and install bcc:  
```shell  
$ cd bcc  
$ mkdir build  
$ cd build  
$ cmake ..  
$ sudo make install  
```
Install go from [here](https://go.dev/doc/install)  
Go back to `netmon/` directory.  
Replace the module path for protos like so:  
```shell  
$ go mod edit -replace github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon=/path/to/netmon/proto  
```
For convenience, `install/` directory has a `setup.sh` script which will set up both `net_helper` and `netmon_main`. It is run like so:  
```shell  
$ bash ./setup_node.sh username@hostname  
```  
   
## Accessing netmon from k3s  
We setup  service routing from the pods running in the k3s cluster to the netmon server running outside using the deployment template in `deployment.yaml`. Basically we set up an endpoint that routes to the service running on the machine's localhost (vis a vis a pod's localhost).   Once you know the names of the nodes in the k3s cluster, create a json file specifying the host names, IPs and ports for netmon. Then run gen_deployment like so:  
```shell  
$ python3 gen_deployment.py -t template.yaml -n hosts.json  
```
After checking that the deployment file for each node is accurate, deploy them into k3s as usual:  
```shell  
$ k3s kubectl apply -f <deployment file name>   
```
