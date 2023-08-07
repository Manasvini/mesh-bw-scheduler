## bw-controller  
This component is responsible for monitoring the state of the cluster and making decisions about pod reschedling. Pretty much, it queries a bunch of services, checks if the bw constraints are being met and potentially deletes a pod.   
### Communication  
The controller talks to the following services:  
- `netmon`: To get info about iperf/traceroute/traffic stats [Note: netmon runs outside k3s]  
- `promethues`: To get stats on inter-pod communication  
- `kube-proxy`: To get information about nodes and pods  
The core logic is in `controller/controller.go` and the rest are stubs gathering data from the above mentioned services.   
### Build and Deployment  
To build for local testing, just run go build like so:   
```shell  
$ go build .  
```
And to run locally, use the default config:  
```shell  
$ ./controller_main  
```
If you want to build a docker image, use the following command:  
```shell  
$ CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .  
```  
To build the actual docker image, run  
```shell  
$ sudo docker build -t bw-controller:latest . 
```   
