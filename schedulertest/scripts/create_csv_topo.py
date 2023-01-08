#!/usr/bin/python3

import json
import pandas as pd
import sys
import argparse 

def create_routing_table(adjList, srcNode, dstNode):
    paths = {src: [] for src in adjList}
    queue = [srcNode]
    paths[srcNode] = []
    visited = {src: False for src in adjList}
    print(srcNode, dstNode)
    while len(queue) > 0:
        curnode = queue.pop(0)
        visited[curnode] = True
        for node in adjList[curnode]:
            if not visited[node] and node != curnode and node not in queue:
                queue.append(node)
                paths[node] = paths[curnode] + [curnode] 
        if visited[dstNode] == True:
            break
    routing_table = []
    if len(paths[dstNode]) > 1:
        routing_table.append({'src':srcNode, 'dst': dstNode, 'next_hop':paths[dstNode][1]})
    elif len(paths[dstNode]) == 1:
        routing_table.append({'src':srcNode, 'dst':dstNode, 'next_hop':dstNode})
    print(routing_table)
    print("\n-----------------------------------\n")
    return routing_table

def read_json(filename: str):
    with open(filename) as ifh:
        data = json.load(ifh)
    return data

def save_links(data, filename):
    links = []
    for src in data:
        for dst in data[src]: 
            links.append({'src':src, 'dst': dst, 'bw_mb':data[src][dst]})
    df = pd.DataFrame(links)
    df.to_csv(filename, index=False)


def save_paths(links, filename):
    all_tables = []
    for src in links:
        for dst  in links:
            routing_table = create_routing_table(links, src, dst)
            all_tables += routing_table
    routing_table_df = pd.DataFrame(all_tables)
    routing_table_df.to_csv(filename, index=False)


def save_nodes(links, cpu, memory, filename):
    nodes = []
    for node in links:
        nodes.append({'nodeId':node, 'cpu':cpu, 'memory_mn':memory})
    df = pd.DataFrame(nodes)
    df.to_csv(filename, index=False)

def parse_args():
    parser = argparse.ArgumentParser(description='Convert json mesh topology to csv files for links, paths and nodes')
    parser.add_argument('-f','--file', help='Topo file', required=True, type=str)
    parser.add_argument('-c','--cpu', help='CPU at node (# cores)', required=True, type=int)
    parser.add_argument('-m','--memory', help='Memory availble at node (MB)', required=True, type=int)
    args = parser.parse_args()
    return args

def main():
    args = parse_args()
    topo_file = args.file
    cpu = args.cpu
    memory = args.memory
    adj_list = read_json(topo_file)
    save_links(adj_list, 'links.csv')
    save_paths(adj_list, 'paths.csv')
    save_nodes(adj_list,  cpu, memory, 'nodes.csv')

if __name__=='__main__':
    main()
