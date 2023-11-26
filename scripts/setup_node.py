import subprocess
import json
import argparse
import time

def setup_node(nodename, username, port, dirname):

    scp = 'scp  -P ' + str(port) 
    host = username + '@' + nodename + ':~/'
    ssh_cmd = 'ssh -p ' + str(port) + ' ' +  username +'@' + nodename 
    mkdir = " mkdir -p mesh-bw-scheduler/containers/"
    ssh_mkdir_cmd = ssh_cmd + mkdir 
    subprocess.run(ssh_mkdir_cmd.split(), check=True,)
    cmd = scp + ' -r ' + dirname + ' ' + host + 'mesh-bw-scheduler/containers/'
    print(cmd)     
    subprocess.run(cmd.split(), check=True,)
    ssh_pip_cmd = ssh_cmd + ' pip3 install -r mesh-bw-scheduler/containers/netmon/net_helper/requirements.txt'
    subprocess.run(ssh_pip_cmd.split(), check=True)

def update_config(nodename, username, port, configdir, nodeid):
    host = username + '@' + nodename + ':~/'
    ssh_cmd = 'ssh -p ' + str(port) + ' ' +  username +'@' + nodename 
    cp_cmd = " cp mesh-bw-scheduler/containers/netmon/netmon_main/" + configdir + "/" + nodeid + ".txt mesh-bw-scheduler/containers/netmon/netmon_main/config_cloudlab.txt"
    ssh_cp_cmd = ssh_cmd + cp_cmd
    print(ssh_cp_cmd)
    subprocess.run(ssh_cp_cmd.split(), check=True)

    cp_cmd = " cp mesh-bw-scheduler/containers/netmon/net_helper/" + configdir + "/" + nodeid + ".json mesh-bw-scheduler/containers/netmon/net_helper/cloudlab_config.json"
    ssh_cp_cmd = ssh_cmd + cp_cmd
    print(ssh_cp_cmd)
    subprocess.run(ssh_cp_cmd.split(), check=True)


 
def read_topo_config(config_filename, user, configdir):
    with open(config_filename) as fh:
        data = json.load(fh)
        dirname = '../containers/netmon'
        for node in data['nodes']:
            print(node)
            time.sleep(10)
            setup_node(node['nodename'],user, node['port'], dirname)
            update_config(node['nodename'], user, node['port'], configdir, node['nodeid'])
  
def parse_args():
    parser = argparse.ArgumentParser(description='Mesh Setup for cloudlab')
    parser.add_argument('-c','--config', help='config file', required=True, type=str)
    parser.add_argument('-u', '--user', help='username', required=True, type=str)
    parser.add_argument('-d', '--configdir', help='dir with node specific config', required=True, type=str)
    args = parser.parse_args()
    return args

def main():
    args = parse_args()
    config_filename = args.config
    read_topo_config(config_filename, args.user, args.configdir)

if __name__=='__main__':
    main()
