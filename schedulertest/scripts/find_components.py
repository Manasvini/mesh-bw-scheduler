import json
import sys

fname = sys.argv[1][:-4]
with open(sys.argv[1]) as fh:
    data = json.load(fh)

def create_graph(data):
    nodes = data['nodes']
    links = data['links']

    g = {}
    for n in nodes:
        g[n['node_id']] = {}

    for l in links:
        if l['dst'] not in g[l['src']]:
            g[l['src']][l['dst']] = 1
    return g

def bfs(graph, start):
    q = [start]
    visited = {}
    while len(q) > 0:   
        node = q.pop(0)
        visited[node] = True
        for n in graph[node]:
              if n not in visited and n not in q:
                    q.append(n)
    return visited


g = create_graph(data)

print('num nodes=', len(g))

all_visited = []
all_nodes = set()
comps = 0
for n in g:
    if n in all_nodes:
        continue
    visited = bfs(g, n)
    comps += 1
    all_visited.append(visited)
    all_nodes = all_nodes.union(set(visited.keys()))

#for idx, comp in enumerate(all_visited):
#    graph = {}
#    for node in comp:
#        graph[node] = g[node]
#    with open( fname +'_' +  str(idx) + '.json', 'w') as fh:
#        json.dump(graph, fh)

print('num comps', comps)
for c in all_visited:
    print(c.keys())
    print(len(c.keys()))

