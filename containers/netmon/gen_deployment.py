import argparse
import yaml
import json

def parse_args():
    ap = argparse.ArgumentParser()
    ap.add_argument('-t', '--template', required=False,
                    help = 'template yaml file', default = 'template.yaml')
    ap.add_argument('-n', '--nodes', required=False,
                    help = 'node hosts file', default='hosts.json')
    args = ap.parse_args()
    return args

def gen_deployment(hostsfile, template):
    with open(template) as fh:
        tplFileGen = yaml.load_all(fh, Loader=yaml.FullLoader)
        docs = []
        for d in tplFileGen:
            docs.append(d)
        with open(hostsfile) as jfh:
            hosts = json.load(jfh)
            for host in hosts['hosts']:
                host_yamls = docs
                # update host name and port in service
                host_yamls[0]['metadata']['name'] = 'netmon-' + host['name']
                host_yamls[0]['spec']['ports'][0]['port'] = host['port']
                # update host name, UP and port in endpoint
                host_yamls[1]['metadata']['name'] = 'netmon-' + host['name']
                host_yamls[1]['subsets'][0]['addresses'][0]['ip'] = host['ip']
                host_yamls[1]['subsets'][0]['ports'][0]['port'] = host['port']
                host_yamls[1]['subsets'][0]['ports'][0]['name'] = 'netmon-' + host['name'] + '-port'
    
                with open(host['name'] + '_deployment.yaml', 'w') as stream:
                    yaml.dump_all(host_yamls, stream)
                print(host, host_yamls)

def main():
    args = parse_args()
    gen_deployment(args.nodes, args.template)

if __name__ == '__main__':
    main()
