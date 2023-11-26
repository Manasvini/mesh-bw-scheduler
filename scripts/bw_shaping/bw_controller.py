import subprocess
import netifaces
import sys
import json
import pandas as pd
import time

ifaces = netifaces.interfaces()

iface_bw_map_file = sys.argv[1]
bw_map = {}

with open(iface_bw_map_file) as fh:
    data = json.load(fh)
    for link in data['links']:
        nodename = link['node']
        iface = link['iface']
        filename = link['bwfile']
        df = pd.read_csv(filename)
        df['5sec_window'] = df['Bitrate'].rolling(5).mean()
        bw_map[iface] = df.dropna()

total_duration = 1800 #seconds

for i in range(0, total_duration, 5): # 5 second increments
    idx = i
    for iface in bw_map:
        if len(bw_map[iface]) < idx:
            continue
        bw_val = bw_map[iface].iloc[idx]['5sec_window']
        print('set bw for ' + iface + ' to ' + str(bw_val) + ' mbit')
        op = 'change'
        if idx == 0:
            op = 'add'
        cmd = 'sudo tc qdisc ' + op + ' dev ' + iface + ' root tbf rate ' + str(bw_val) + 'mbit' +  ' latency 0.1ms burst 500kb mtu 10000'
        print(cmd)
        subprocess.run(cmd.split(), check=True)
        time.sleep(5) # wait 5 sec before next bw val
        
for iface in bw_map:
    cmd = 'sudo tc qdisc del dev ' + iface + ' root'
    subprocess.run(cmd.split(),  check=True)
