# Cluster Setup  
This doc walks through the steps for setting up a k3s cluster, PION SFU, and Prometheus for metric collection.  
## k3s setup  
First get the latest release of k3s from [here](https://github.com/k3s-io/k3s/releases/) like so:  
```shell  
$ wget https://github.com/k3s-io/k3s/releases/download/v1.24.4%2Bk3s1/k3s  
```  
Make k3s executable  
```shell  
$ chmod +x k3s  
```  
So we now have the server up and running. You can check and see the nodes in the cluster  
```shell  
$ sudo ./k3s kubectl get nodes  
```  
It should list at least one node.  
Now if you want to add nodes to the cluster, similarly install k3s on the other nodes. To start a node as a k3s agent, run  
```shell  
$ sudo ./k3s agent --server https://server_ip:6443 --token [k3s_token]  
```  
You can find the k3s server token in `/var/lib/rancher/k3s/server/node-token`. Et voila! We have a 2 node cluster. This can be verified by running the nodes command again.  
```shell  
$ sudo ./k3s kubectl get nodes  
```  
## PION Setup  
We need to create a service deployment yaml from the [docker-compose.yml](ion/ion-docker-compose.yml) for PION (Check `ion/` directory). We use [kompose](https://github.com/kubernetes/kompose) to do this. First download the kompose binary like so:  
```shell  
$ curl -L https://github.com/kubernetes/kompose/releases/download/v1.26.1/kompose-linux-amd64 -o kompose  
$ chmod +x kompose  
```  
Next, feed PION's docker-compose.yml to kompose.   
 ```shell  
$ ./kompose convert -f ion/ion-docker-compose.yml  
```  
This should produce the `sfu-service.yaml` and `sfu-deployment.yaml` files. We can now deploy the service onto our cluster.  
```shell  
$ ./k3s kubectl apply -f sfu-deployment.yaml,sfu-service.yaml  
```  
## Prometheus Setup  
We use the `kube-prometheus-stack` which packages node-exporter, grafana and prometheus-operator for deploying onto a k8s cluster. We don't particularly need grafana right now, so I've disabled it in the config. We don't use NFS for storage, jsut temporary store, so metrics don't persist. More on this later. First we install helm which is the package manager for k3s.  
```shell  
$ wget https://get.helm.sh/helm-v3.9.4-linux-amd64.tar.gz   
$ tar -xzvf helm-v3.9.3-linux-amd64.tar.gz  
$ chmod +x helm/helm  
```  
Next, add kube-prometheus-stack chart to helm.  
```shell  
$ helm repo add prometheus-community https://prometheus-community.github.io/helm-charts   
$ helm update  
```
NOTE: You might have to do the above steps for helm with sudo if you're running k3s under sudo.  
We next have to install `kube-prometheus-stack`. But before that, a word on how we want the installation to work. Check out [values.yaml](prometheus-install/values.yaml) to verify that grafana is not being installed. In other words, `grafana.enabled` is false. We've also for the moment not enabled any storage for metrics, so we're just using temporary storage. So, `storageSpec.emptyDir` is set to `Memory`. We also need to specify the address of `kube-scheduler` and `kube-apiserver` so that Prometheus can scrape metrics from them. These configs are specified in [prometheus.yaml](prometheus-install/prometheus.yaml). Note that we're exposing Prometheus metrics on `localhost:9090`.  
To install Prometheus, we run  
```shell  
$ helm install --values prometheus-install/values.yaml -f prometheus-install/prometheus.yaml -n monitoring monitoring prometheus-community/kube-prometheus-stack --kubeconfig /etc/rancher/k3s/k3s.yaml
```     
By default, node exporter (which exposes node metrics) is enabled, and exposes metrics in port 9100. Prometheus in turn scrapes these metrics from all the nodes and exposes them on port 9090.  
That's all! We're done with the setup.  

