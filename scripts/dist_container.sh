#!/bin/bash

git pull

echo "Creating image on master"
scp -r ~/mesh/mesh-bw-scheduler/containers/$1 cvuser@cv2:~/mesh/tmp/
scp -r ~/mesh/mesh-bw-scheduler/scripts/ cvuser@cv2:~/mesh/tmp/scripts/

echo `cat ~/passfile` | ssh -tt cvuser@cv2 "bash ~/mesh/tmp/scripts/build_container.sh $1" 
echo `cat ~/passfile` | ssh -tt cvuser@cv2 "sudo ~/mesh/k3s ctr images import ~/mesh/tmp/$1.tar"
echo `cat ~/passfile` | ssh -tt cvuser@cv2 "sudo chmod +777 ~/mesh/tmp/$1.tar"
scp cvuser@cv2:~/mesh/tmp/$1.tar ~/mesh/tmp/

echo "Deleting residuals from master"
ssh cvuser@cv2 -T "rm -rf ~/mesh/tmp/scripts"
ssh cvuser@cv2 -T "rm -rf ~/mesh/tmp/$1"
ssh cvuser@cv2 -T "rm ~/mesh/tmp/$1.tar"

for i in 1
do
    echo "Running script on cv$i"
    scp ~/mesh/tmp/$1.tar cvuser@cv$i:~/mesh/tmp/
    echo `cat ~/passfile` | ssh -tt cvuser@cv$i "sudo ~/mesh/k3s ctr images import ~/mesh/tmp/$1.tar"
    ssh cvuser@cv$i -T "rm -rf ~/mesh/tmp/$1.tar"
done

rm ~/mesh/tmp/$1.tar
echo "DONE"