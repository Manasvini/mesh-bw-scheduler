hostname=$1
ssh $hostname 'mkdir netmon'
scp -r ../proto $hostname:~/netmon/
scp -r ../net_helper $hostname:~/netmon/
scp -r ../netmon_main $hostname:~/netmon/
scp clang_libs.cmake $hostname:~/netmon/
scp install.sh $hostname:~/netmon/
ssh $hostname 'cd netmon/net_helper; pip3 install -r requirements.txt'
ssh $hostname 'cd netmon && bash ./install.sh'



