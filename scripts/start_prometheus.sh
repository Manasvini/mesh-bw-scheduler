cd ~/mesh/mesh-bw-scheduler

git pull

scp -r ~/mesh/mesh-bw-scheduler/prometheus cvuser@cv2:~/mesh/tmp/prometheus


echo `cat ~/passfile` | ssh -tt cvuser@cv2 "sudo ~/mesh/helm install --values ~/mesh/tmp/prometheus/prometheus-install/values.yaml -f ~/mesh/tmp/prometheus/prometheus-install/prometheus.yaml -n monitoring monitoring prometheus-community/kube-prometheus-stack --kubeconfig /etc/rancher/k3s/k3s.yaml" 

echo `cat ~/passfile` | ssh -tt cvuser@cv2 "rm -rf ~/mesh/tmp/prometheus" 

