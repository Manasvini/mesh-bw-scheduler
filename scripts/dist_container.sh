scp -r ~/mesh/mesh-bw-scheduler/containers/$1 cvuser@cv2:~/mesh/tmp/
scp -r ~/mesh/mesh-bw-scheduler/scripts/ cvuser@cv2:~/mesh/tmp/scripts/

ssh cvuser@cv2 -t "~/mesh/tmp/scripts/build_container.sh $1"

ssh cvuser@cv2 -t "rm -rf /home/cvuser/mesh/tmp/scripts"
ssh cvuser@cv2 -t "rm -rf /home/cvuser/mesh/tmp/$1"