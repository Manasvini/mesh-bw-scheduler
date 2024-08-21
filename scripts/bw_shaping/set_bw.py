import subprocess
import json
import argparse
import time

def setup_node(nodename, username, port, bw_files, bw_controller_file, bw_config_file):

    scp = 'scp  -P ' + str(port) 
    host = username + '@' + nodename + ':~/'
    cmd = scp + ' ' + bw_controller_file + ' ' + host
    print(cmd)
    ssh_cmd = 'ssh -p ' + str(port) + ' ' +  username +'@' + nodename 
    mkdir = " mkdir -p measurements"
    ssh_cmd += mkdir 
    print(ssh_cmd)
    subprocess.run(ssh_cmd.split(),   check=True,)
    subprocess.run(cmd.split(), check=True, )
    cmd = scp + ' ' + bw_config_file + ' ' +  host
    subprocess.run(cmd.split(), check=True, )
    for f in bw_files:
        cmd = scp + ' ' + f +' ' +  host + 'measurements/'
        print(cmd)
        subprocess.run(cmd.split(), check=True,)
        

def read_topo_config(config_filename, user):
    with open(config_filename) as fh:
        data = json.load(fh)
        for node in data['nodes']:
            if int(node['port'])<25011:
                continue
            with open(node['bw_config_file']) as fh:
                bw_file_data = json.load(fh)
                files = []
                for l in bw_file_data['links']:
                    files.append(l['bwfile'])
            time.sleep(10)
            setup_node(node['nodename'],user, node['port'], files, 'bw_controller.py', node['bw_config_file'])

  
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
