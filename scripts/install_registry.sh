echo "Intalling registry image"
sudo docker image pull registry

echo "Run the registry"
docker run -d -p 5000:5000 --restart=always --name registry registry

echo "Use the following command to get the images in registry"
echo "curl -X GET 192.168.160.23:5000/v2/_catalog"
echo "List of images in current registry"
curl -X GET 192.168.160.23:5000/v2/_catalog

