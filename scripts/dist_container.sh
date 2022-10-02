#!/bin/bash

echo "Creating image on master"
scp -r ~/mesh/mesh-bw-scheduler/containers/$1 cvuser@cv2:~/mesh/tmp/
scp -r ~/mesh/mesh-bw-scheduler/scripts/ cvuser@cv2:~/mesh/tmp/scripts/

ssh cvuser@cv2 -t "bash ~/mesh/tmp/scripts/build_container.sh $1" <<< $2
ssh cvuser@cv2 -t "sudo ~/mesh/k3s ctr images import ~/mesh/tmp/$1.tar" <<< $2
ssh cvuser@cv2 -t "sudo chmod +777 ~/mesh/tmp/$1.tar" <<< $2
scp cvuser@cv2:~/mesh/tmp/$1.tar ~/mesh/tmp/

echo "Deleting residuals from master"
ssh cvuser@cv2 -t "rm -rf ~/mesh/tmp/scripts" <<< $2
ssh cvuser@cv2 -t "rm -rf ~/mesh/tmp/$1" <<< $2
ssh cvuser@cv2 -t "rm ~/mesh/tmp/$1.tar" <<< $2

for i in 1
do
    echo "Running script on cv$i"
    scp ~/mesh/tmp/$1.tar cvuser@cv$i:~/mesh/tmp/
    ssh cvuser@cv$i -t "sudo ~/mesh/k3s ctr images import ~/mesh/tmp/$1.tar" <<< $2
    ssh cvuser@cv$i -t "rm -rf ~/mesh/tmp/$1.tar" <<< $2
done

rm ~/mesh/tmp/$1.tar
echo "DONE"