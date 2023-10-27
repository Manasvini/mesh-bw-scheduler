import subprocess
import json
import argparse
import time

def setup_node(nodename, username, port, dirname):

    scp = 'scp  -P ' + str(port) 
    host = username + '@' + nodename + ':~/'
    ssh_cmd = 'ssh -p ' + str(port) + ' ' +  username +'@' + nodename 
    mkdir = " mkdir -p mesh-bw-scheduler/containers/"
    ssh_cmd += mkdir 
    subprocess.run(ssh_cmd.split(), check=True,)
    cmd = scp + ' -r ' + dirname + ' ' + host + 'mesh-bw-scheduler/containers/'
    print(cmd)     
    subprocess.run(cmd.split(), check=True,)

def read_topo_config(config_filename, user):
    with open(config_filename) as fh:
        data = json.load(fh)
        dirname = '../containers/netmon'
        for node in data['nodes']:
            time.sleep(10)
            setup_node(node['nodename'],user, node['port'], dirname)

  
def parse_args():
    parser = argparse.ArgumentParser(description='Mesh Setup for cloudlab')
    parser.add_argument('-c','--config', help='config file', required=True, type=str)
    parser.add_argument('-u', '--user', help='username', required=True, type=str)
    args = parser.parse_args()
    return args

def main():
    args = parse_args()
    config_filename = args.config
    read_topo_config(config_filename, args.user)

if __name__=='__main__':
    main()
