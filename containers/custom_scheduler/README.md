## Custom Scheduler  
This scheduler reads annotations from the deployment.yaml file for a container, collects dependencies and then creates a pod graph which is then topologically sorted before applyng scheduling policies.   
## Build  
There are two options:  
1. Build and run outside k3s cluster but still make calls to kube API  
2. Run inside k3s cluster  
#### Standalone build  
```shell  
$ go build .  
```  
#### Docker build for running in k3s cluster  
```shell  
$  CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .    
```  
Make sure that the Dockerfile refers to the appropriate config file you want to use for the cluster build. After the binary is built, we need to create a docker image and push it to the registry so that it can be accessed from any node.  
```shell  
$ sudo docker build -t custom_scheduler:latest .  
$ sudo docker tag custom_scheduler:latest manasvini1/custom_scheduler:latest 
$ sudo docker push manasvini1/custom_scheduler:latest  
```  
## Run  
If running locally, run this command:  
```shell  
$ ./custom_scheduler -config <config_file.json>  
```  
If the scheduler is running inside the k3s cluster, make sure that the namespace exists:  
```shell  
$ sudo k3s kubectl create namespace epl  
$ sudo k3s apply -f deployment.yaml  
```  

