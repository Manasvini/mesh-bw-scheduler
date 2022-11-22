
import subprocess
import sys
import json

def print_cmd(cmd_args):
    print('cmd is ', ' '.join(cmd_args))

def install_server(server_hostname, ssh_username, ssh_config_path):
    print(' install k3s on server', server_hostname, 'ssh user: ', ssh_username, 'ssh cfg:', ssh_config_path)
    print_cmd(['ssh',  '-F', ssh_config_path, server_hostname, '"mkdir cluster_install"'])

    result = subprocess.call(['ssh',   '-F', ssh_config_path, server_hostname,  '/bin/mkdir cluster_install'], shell=False, stdin=sys.stdin, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    print(result)
    print_cmd(['scp',  '-F', ssh_config_path, 'init_k3s_server.sh', ssh_username + '@' +server_hostname + ':~/cluster_install/'])
    result = subprocess.run(['scp',  '-F', ssh_config_path, 'init_k3s_server.sh', ssh_username + '@' +server_hostname + ':~/cluster_install/'], capture_output=True, text=True ) 
    print(result.stdout)
    print(result.stderr)
    print_cmd(['ssh', server_hostname, '-F', ssh_config_path, 'cd  ./cluster_install && bash init_k3s_server.sh'])
    result = subprocess.run(['ssh', server_hostname, '-F', ssh_config_path, 'cd  ./cluster_install && bash init_k3s_server.sh'], capture_output=True, text=True)
    print(result.stdout)
    print_cmd(['scp',  '-F', ssh_config_path, ssh_username + '@' +server_hostname + ':~/cluster_install/token.txt', './'])
    result = subprocess.run(['scp',  '-F', ssh_config_path, ssh_username + '@' +server_hostname + ':~/cluster_install/token.txt', './'], capture_output=True, text=True ) 
    print(result.stdout)
    
def install_agent(agent_hostname, ssh_username, ssh_config_path, token, server_ip):
    print(' install k3s agent', agent_hostname, 'ssh user: ', ssh_username, 'ssh cfg:', ssh_config_path, 'server ', server_ip, 'token ', token)
    print_cmd(['ssh',  '-o', 'StrictHostKeyChecking=no', agent_hostname, '-F', ssh_config_path, '"mkdir cluster_install"'])
    result = subprocess.run(['ssh',  '-o', 'StrictHostKeyChecking=no', agent_hostname,  '-F', ssh_config_path, 'mkdir cluster_install'], capture_output=True, text=True)
    print(result.stdout)
    print_cmd(['scp',  '-F', ssh_config_path, 'init_k3s_agent.sh', ssh_username + '@' + agent_hostname + ':~/cluster_install/'])
    result = subprocess.run(['scp',  '-F', ssh_config_path, 'init_k3s_agent.sh', ssh_username + '@' + agent_hostname + ':~/cluster_install/'], capture_output=True, text=True ) 
    print(result.stdout)
    print(result.stderr)
    cmd = 'bash init_k3s_agent.sh ' + server_ip + ' ' + token 
    print_cmd(['ssh', agent_hostname, '-F', ssh_config_path, 'cd  ./cluster_install && ' + cmd  ])
    result = subprocess.run(['ssh', agent_hostname, '-F', ssh_config_path, 'cd  ./cluster_install && ' + cmd  ], capture_output=True, text=True)
    print(result.stdout)

def install_k3s(k3s_config_file):
    with open(k3s_config_file) as fh:
        data = json.load(fh)
        server_hostname = data['server_hostname']
        agents = data['agents']
        server_ip = data['server_ip']
        ssh_username = data['ssh_username']
        ssh_cfg = data['ssh_config']
        #install_server(server_hostname, ssh_username, ssh_cfg)
        token = ''
        with open('token.txt') as f:
            token = f.read()
        for agent in agents:
            install_agent(agent, ssh_username, ssh_cfg, token, server_ip)
        
def main():
    k3scfg = sys.argv[1]
    install_k3s(k3scfg)

if __name__=='__main__':
    main()
