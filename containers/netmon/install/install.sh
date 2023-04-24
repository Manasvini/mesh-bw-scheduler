cd

wget https://go.dev/dl/go1.20.3.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.20.3.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin >> ~/.bashrc

source ~/.bashrc 

sudo apt install -y bison build-essential cmake flex git libedit-dev \
  llvm-10  llvm10.0-dev clang-10 libclang-10.0-dev python zlib1g-dev libelf-dev libfl-dev python3-setuptools
git clone --recurse-submodules  https://github.com/iovisor/bcc.git

export LLVM_ROOT=/usr/lib/llvm-10

mv ~/netmon/clang_libs.cmake ~/bcc/cmake/

cd ~/bcc

mkdir build
cd build

cmake ..
make
sudo make install
#sudo apt-get install -y bpfcc-tools linux-headers-$(uname -r)

cd ~/netmon/netmon_main
go mod edit -replace github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon=/home/cvuser/netmon/proto
go mod tidy
go build .
