sudo apt update
sudo apt install -y python3-pip
pip3 install  -U  multidict>=1.2.2 asyncio typing-extensions attrs yarl async_timeout idna_ssl charset-normalizer aiosignal aiohttp

sudo apt install -y libssl-dev libz-dev luarocks
sudo luarocks install liasocket

sudo apt install -y  ca-certificates curl  gnupg lsb-release


sudo apt install -y apt-transport-https ca-certificates curl software-properties-common

sudo mkdir -p /etc/apt/keyrings

curl -fsSL https://download.docker.com/linux/ubuntu/gpg |  sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo   "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
$(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt update
sudo apt install -y  docker-ce docker-ce-cli containerd.io docker-compose-plugin

