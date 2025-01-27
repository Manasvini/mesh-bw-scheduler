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
import time
import random
from logging.handlers import RotatingFileHandler
import logging

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


def run_iperf(host, bwlimit=None):
    client = iperf3.Client()
    client.duration = 2
    client.server_hostname = host
    client.port = config['node'] + 5201
    client.protocol = 'tcp'
    time.sleep(random.randint(1, 10))
    if bwlimit is not None:
        print(type(bwlimit), bwlimit)
        #client.protocol = 'udp'
        client.bandwidth = int(float(bwlimit))
        #client.reverse = True
    res = client.run()
    print(host, bwlimit, res.sent_bps)
    #res.json
    if 'error' in res.json:
        return res.json
    
    #if bwlimit is not None:
    #    return {'host':host, 'snd': res.json['end']['streams'][0]['udp']['bits_per_second'], 'rcv':res.json['end']['streams'][0]['udp']['bits_per_second']}
    app.logger.error('sent ' + str(res.sent_bytes) + ' to ' +  host)
    return {'Host':host, 'SndBw': res.sent_bps, 'RcvBw':res.received_bps} #, 'Snd':res.sent_bytes, 'Rcv':res.received_bytes}


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
    app.logger.error('Got bw req for host ' + request.args.get('host')) 
    hostname = request.args.get('host')

    #iperf_hosts = get_hosts()
    bwlimit = request.args.get('bwmax')
    print(hostname, bwlimit)
    #results = [run_iperf(hostname, bwlimit)]
    #print(len(results))
    final_results = []
    elapsed = 0
    while elapsed < 60:
        res = run_iperf(hostname, bwlimit)
        if 'error' in res:
            sleep_time = random.randint(0, 5)
            time.sleep(sleep_time)
            elapsed += sleep_time
        else:
            final_results.append(res)
            break
    
    print(final_results)
    return json.dumps({'bandwidthResults':final_results})




def run_ping(host):
    result = subprocess.check_output(['ping', '-c',  '10', host])
    print(result)
    vals = str(result).split('\n')
    stats = vals[-1].split()[-2].split('/')
    avg_latency = stats[1]
    latency = float(avg_latency)
    unit = vals[-1].split()[-1]
    if unit == 's':
        return latency * 1e3
    if unit == 'us':
        return latency / 1e3
    return latency

@app.route('/latency')
def get_ping():
    hostname = request.args.get('host')
    print(hostname)
    latency = run_ping(hostname)
    results = [{'host': hostname, 'latency':latency}]
    return json.dumps({'latencyResults':results})

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
    file_handler = RotatingFileHandler('python.log', maxBytes=1024 * 1024, backupCount=20)
    file_handler.setLevel(logging.INFO)
    formatter = logging.Formatter("%(asctime)s - %(name)s - %(levelname)s - %(message)s")
    file_handler.setFormatter(formatter)
    app.logger.addHandler(file_handler) 
    app.run(debug=True, host=args.ip, port=port)

