import json
import sys

def read_graph(filename):
    with open(filename) as fh:
        data = json.load(fh)
    return data

def create_subgraph(num_nodes, graph, min_node_degree=1):
    if len(graph) == num_nodes:
        return graph

    nodes_to_remove = []
    node_degree = min_node_degree
    for node in graph:
        if len(graph[node]) <= node_degree:
            nodes_to_remove.append(node)
        if len(graph) - len(nodes_to_remove) == num_nodes:
            break 
    subgraph = {}
    for node in graph:
        if node not in nodes_to_remove:
            subgraph[node] = {}
            for link in graph[node]:
                if link not in nodes_to_remove:
                    subgraph[node][link] = graph[node][link]
    return subgraph

def main():
    inputfile = sys.argv[1]
    outputfile = sys.argv[2]
    graph = read_graph(inputfile)
    subgraph = create_subgraph(len(graph)-10, graph, 2)
    with open(outputfile, 'w') as fh:
        json.dump(subgraph, fh)

if __name__=='__main__':
    main()    
