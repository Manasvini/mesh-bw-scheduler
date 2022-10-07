# Cluster Setup  
This doc walks through the steps for setting up a k3s cluster, PION SFU, and Prometheus for metric collection.  
## k3s setup  
First get the latest release of k3s from [here](https://github.com/k3s-io/k3s/releases/) like so:  
```shell  
wget https://github.com/k3s-io/k3s/releases/download/v1.24.4%2Bk3s1/k3s  
```  
Make k3s executable  
```shell  
chmod +x k3s  
```  
Start the server
```shell
sudo ./k3s server
```
So we now have the server up and running. You can check and see the nodes in the cluster  
```shell  
sudo ./k3s kubectl get nodes  
```  
It should list at least one node.  
Now if you want to add nodes to the cluster, similarly install k3s on the other nodes. To start a node as a k3s agent, run  
```shell  
sudo cat /var/lib/rancher/k3s/server/node-token
sudo ./k3s agent --server https://server_ip:6443 --token [k3s_token]  
```  
You can find the k3s server token in `/var/lib/rancher/k3s/server/node-token`. Et voila! We have a 2 node cluster. This can be verified by running the nodes command again. 
Files regarding k3s are stored in `/var/lib/rancher/k3s`.
```shell  
sudo ./k3s kubectl get nodes  
```  

Get list of running pods on k3s
```shell
sudo ./k3s  kubectl get pods --all-namespaces
```

Kill k3s instances
```
ps -aux | grep k3 | awk '{ print $2 }' | while IFS= read -r line; do sudo kill -9 $line; done
```
## PION Setup  
We need to create a service deployment yaml from the [docker-compose.yml](ion/ion-docker-compose.yml) for PION (Check `ion/` directory). We use [kompose](https://github.com/kubernetes/kompose) to do this. First download the kompose binary like so:  
```shell  
curl -L https://github.com/kubernetes/kompose/releases/download/v1.26.1/kompose-linux-amd64 -o kompose  
chmod +x kompose  
```  
Next, feed PION's docker-compose.yml to kompose.   
 ```shell  
./kompose convert -f mesh-bw-scheduler/ion/ion-docker-compose.yml
```  
This should produce the `sfu-service.yaml` and `sfu-deployment.yaml` files. We can now deploy the service onto our cluster.  
```shell  
./k3s kubectl apply -f sfu-deployment.yaml,sfu-service.yaml  
```  
## Prometheus Setup  
We use the `kube-prometheus-stack` which packages node-exporter, grafana and prometheus-operator for deploying onto a k8s cluster. We don't particularly need grafana right now, so I've disabled it in the config. We don't use NFS for storage, jsut temporary store, so metrics don't persist. More on this later. First we install helm which is the package manager for k3s.  
```shell  
wget https://get.helm.sh/helm-v3.9.4-linux-amd64.tar.gz   
tar -xzvf helm-v3.9.4-linux-amd64.tar.gz 
mv linux-amd64/helm ./helm 
chmod +x helm
```  
Next, add kube-prometheus-stack chart to helm.  
```shell  
sudo ./helm repo add prometheus-community https://prometheus-community.github.io/helm-charts  
```
NOTE: You might have to do the above steps for helm with sudo if you're running k3s under sudo.  
We next have to install `kube-prometheus-stack`. But before that, a word on how we want the installation to work. Check out [values.yaml](prometheus-install/values.yaml) to verify that grafana is not being installed. In other words, `grafana.enabled` is false. We've also for the moment not enabled any storage for metrics, so we're just using temporary storage. So, `storageSpec.emptyDir` is set to `Memory`. We also need to specify the address of `kube-scheduler` and `kube-apiserver` so that Prometheus can scrape metrics from them. These configs are specified in [prometheus.yaml](prometheus-install/prometheus.yaml). Note that we're exposing Prometheus metrics on `localhost:9090`.  
To install Prometheus, we run  
```shell  
sudo ./k3s kubectl create namespace monitoring  
sudo ./helm install --values ./mesh-bw-scheduler/prometheus-install/values.yaml -f ./mesh-bw-scheduler/prometheus-install/prometheus.yaml -n monitoring monitoring prometheus-community/kube-prometheus-stack --kubeconfig /etc/rancher/k3s/k3s.yaml
```     
Check if prometheus is running as expected.  
```shell  
sudo ./k3s kubectl get pods -n monitoring  
```  

Expose this service
```
sudo ./k3s kubectl port-forward prometheus-monitoring-kube-prometheus-prometheus-0 9091:9090 -n monitoring
```
Make sure that `monitoring-kube-prometheus-prometheus` status is RUNNING. 
By default, node exporter (which exposes node metrics) is enabled, and exposes metrics in port 9100. Prometheus in turn scrapes these metrics from all the nodes and exposes them on port 9090.  
That's all! We're done with the setup.  

## Test the setup

Get services in the monitoring namespace
```shell
sudo ./k3s kubectl get services -n monitoring
```

Run a curl command to the ip address of `monitoring-kube-prometheus-prometheus` and it should give a succesfull output.
```shell
curl 'http://<ip>:9090/api/v1/query?query=up'
```

## Master setup

Master is on cv2 with ip `192.168.160.23`.

First install docker using the `install_docker.sh` script.

First install docker registry using the `install_registry.sh` script.

### Creating docker images

Go to a folder with a Dockerfile and do the following
```shell
sudo docker image build -t <local_image_name> .
```

### Master registry updates

Images can be listed from registry using the following command
```shell
curl -X GET 192.168.160.23:5000/v2/_catalog
```

Tag new images using the following command on master
```shell
sudo docker tag <local_image> localhost:5000/<name_in_registry>
sudo docker push localhost:5000/<name_in_registry>
```

Now remove the local images
```shell
sudo docker image remove <local_image> 
sudo docker image remove localhost:5000/<name_in_registry>
```

Now try pulling the image from any machine
```shell
sudo docker image pull 192.168.160.23:5000/simple_flask_container
```

### Connecting to docker registry from any machine

Enable registry http calls by adding the `{"insecure-registries":["192.168.160.23:5000"]}` into `/etc/docker/daemon.json`. Also add ` DOCKER_OPTS="--config-file=/etc/docker/daemon.json"` to `/etc/default/docker`

Restart docker service
```shell
sudo systemctl stop docker
sudo systemctl start docker
```

Now create a container from the image
```shell
sudo docker run -p 6000:6000 -d 192.168.160.23:5000/<image_name>
```

## Running a kubernetes YAML

```shell
sudo ./k3s kubectl apply -f simple_flask.yaml
```

```shell
sudo ./k3s kubectl get deployments
sudo ./k3s kubectl get services
```

Delete after done
```shell
sudo ./k3s kubectl delete -f simple_flask.yaml
```

## Copy the container from docker into containerd

Create the tar from the dockerfile. Run these commands from the Dockerfile directory.
```shell
sudo docker build -t <image_name:version> .
sudo docker save --output <tar_file> <image_name:version>
```

Copy the tar to ALL pods
```shell
scp simple_flask_container.tar cv2:/home/cvuser/mesh/simple_flask_container.tar
```

Import the tar
```shell
sudo ./k3s ctr images import simple_flask_container.tar
```

### Assorted kubectl

Remove annotations
```shell
sudo ./k3s kubectl annotate node cv1 prometheus.io/scrape-
```


```shell
ssh -N cvuser@192.168.160.23 -L 8002:localhost:8001
```
