#!/usr/bin/python3

import argparse
import sys
import subprocess
import os
import pandas as pd

def run_scheduler(input_dirname, num_trials, result_dir, scheduler_type)
    cmd = ['./schedulertest', '-i', input_dirname, '-s', scheduler_type]
    times = []

    app_df = pd.read_csv(input_dirname + '/app.csv')
    num_comps = len(app_df)

    node_df = pd.read_csv(input_dirname + '/nodes.csv')
    num_nodes = len(node_df)

    
    deps_df = pd.read_csv(input_dirname + '/deps.csv')
    num_deps = len(deps_df)
 
    for i in range(int(num_trials)):    
        p = subprocess.run(cmd, capture_output=True, text=True)
        print(i)
        #print(p)
        lines = str(p.stdout).split('\n')
        time_millis = float(lines[-1].split(' ')[2])
        times.append(time_millis)
    
    os.makedirs(result_dir, exist_ok=True)
    df = pd.DataFrame({'runtime_ms': times})
    df['num_nodes'] = num_nodes
    df['num_components'] = num_comps
    df['num_deps'] = num_deps
    df.to_csv(result_dir + '/' + 'comp_' + str(num_comps) + '_nodes_' + str(num_nodes) + '.csv', index=False)         
    
def parse_args():
    parser = argparse.ArgumentParser(description='time trial for scheduler')
    parser.add_argument('-i', '--input_dir', help='input directory for scheduler')
    parser.add_argument('-o', '--output_dir', help='output directory')
    parser.add_argument('-n', '--num_trials', help='num trials')

    parser.add_argument('-t', '--scheduler_type', help='scheduler type(optimal/maxbw)')


    args = parser.parse_args()
    return args

def main():
    args = parse_args()
    print(args)
    run_scheduler(args.input_dir, args.num_trials, args.output_dir, args.scheduler_type)

if __name__ == '__main__':
    main()
