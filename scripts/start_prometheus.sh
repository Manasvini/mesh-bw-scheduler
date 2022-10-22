cd ~/mesh/mesh-bw-scheduler

git pull

scp -r ~/mesh/mesh-bw-scheduler/other cvuser@cv2:~/mesh/tmp/other

sudo ./helm install --values ~/mesh/tmp/other/prometheus-install/values.yaml -f ~/mesh/tmp/other/prometheus-install/prometheus.yaml -n monitoring monitoring prometheus-community/kube-prometheus-stack --kubeconfig /etc/rancher/k3s/k3s.yaml

echo `cat ~/passfile` | ssh -tt cvuser@cv2 "sudo ./helm install --values ~/mesh/tmp/other/prometheus-install/values.yaml -f ~/mesh/tmp/other/prometheus-install/prometheus.yaml -n monitoring monitoring prometheus-community/kube-prometheus-stack --kubeconfig /etc/rancher/k3s/k3s.yaml" 

echo `cat ~/passfile` | ssh -tt cvuser@cv2 "rm -rf ~/mesh/tmp/other" 

