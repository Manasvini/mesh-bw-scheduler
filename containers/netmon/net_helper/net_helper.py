from flask import Flask
import os
import iperf3
import nmap
import json
import argparse 
import ifcfg 
import subprocess
from concurrent.futures import ThreadPoolExecutor, as_completed, wait, ALL_COMPLETED
from flask import request

app = Flask(__name__)

def get_config(filename):
    with open(filename) as fh:
        return json.load(fh)


def run_nmap(subnet):
    print('subnet is ', subnet)
    nm = nmap.PortScanner()
    nm.scan(hosts=subnet, arguments='-sn')
    hosts_list = [(x, nm[x]['status']['state']) for x in nm.all_hosts()]
    print(hosts_list)
    return hosts_list


def run_iperf(host):
    client = iperf3.Client()
    client.duration = 5
    client.server_hostname = host
    client.port = 5201
    res = client.run()
    print(res.json)
    if 'error' in res.json:
        return res.json
    return {'host':host, 'snd': res.json['end']['streams'][0]['sender']['bits_per_second'], 'rcv':res.json['end']['streams'][0]['receiver']['bits_per_second']}


@app.route('/traceroute')
def get_traceroute():
    hosts = config['hosts']
    routes = []
    for host in hosts:
        proc = subprocess.run(['traceroute', host], stdout=subprocess.PIPE)

        vals = str(proc.stdout).replace("'", '').split('\\n ')
        print(vals)
        if len(vals) < 2:
            continue
        hops = vals[1:]
        route = []
        
        for h in hops:
            ip = h.split()[2].replace('(', '').replace(')', '')
            
            route.append(ip)
        routes.append({'host':host, 'route':route})
    return json.dumps({'tracerouteResults':routes})


def get_hosts():
    results = []
    intfs = config['interfaces']
    all_ifaces = ifcfg.interfaces().items()
    iperf_hosts = []
    for i in intfs:
        for dev, iface in all_ifaces:
            if i == dev:
                dev_ip = iface['inet']
                print(dev_ip)
                dev_net = dev_ip.split('.')[:3] + ['0/24']
                print(dev_net)
                hosts = run_nmap('.'.join(dev_net))
                print(hosts)
                for host, status in hosts:
                    if status != 'up':
                        continue
                    iperf_hosts.append(host)
                #    print(host, status)
                #    res = run_iperf(host)
    return iperf_hosts

@app.route('/bw')
def get_bw():
    hostname = request.args.get('host')
    #iperf_hosts = get_hosts()
    print(hostname)
    results = [run_iperf(hostname)]
    print(len(results))
    final_results = []
    for res in results:
        print(res)
        if 'error' in res:
            continue
        final_results.append({'host':res['host'], 'snd':res['snd'], 'rcv':res['rcv']})
    print(final_results)
    return json.dumps({'bandwidthResults':final_results})



def parse_args():
    ap = argparse.ArgumentParser()
    ap.add_argument('-c', '--config', required=False,
                    help = 'path to config file', default = './config.json')
    ap.add_argument('-i', '--ip', required=False,
                    help = 'ip address of host', default = '0.0.0.0')
    return ap.parse_args()    

if __name__ == "__main__":
    args = parse_args()
    port = int(os.environ.get('PORT', 6000))
    config = get_config(args.config)
    app.run(debug=True, host=args.ip, port=port)

