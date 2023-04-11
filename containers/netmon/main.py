from flask import Flask
import os
import iperf3
import nmap
import json
import argparse 
import ifcfg 

nm = nmap.PortScanner()
app = Flask(__name__)

def get_config(filename):
    with open(filename) as fh:
        return json.load(fh)

config = get_config()

def run_nmap(subnet):
    nm.scan(hosts=subnet, arguments='-sn')
    hosts_list = [(x, nm[x]['status']['state']) for x in nm.all_hosts()]
    print(hosts_list)
    return hosts_list


def run_iperf(host):
    client = iperf3.Client()
    client.duration = 15
    client.server_hostname = host
    client.port = 5201
    return client.run()

@app.route('/bw')
def get_bw():
    results = []
    intfs = config['interfaces']
    all_ifaces = ifcfg.interfaces().items()
    for i in intfs:
        for dev, iface in all_ifaces:
            if i == dev:
                dev_ip = interface['inet']
                hosts = run_nmap(dev_ip)
                for host, status in hosts:
                    print(host, status)
                    results.append({'host':host, 'result':run_iperf(host)})
    return json.loads(results)

def parse_args():
    ap = argparse.ArgumentParser()
    ap.add_argument('-c', '--config', required=False,
                    help = 'path to config file', default = './config.json')
    return args    

if __name__ == "__main__":
    port = int(os.environ.get('PORT', 6000))
    app.run(debug=True, host='0.0.0.0', port=port)

