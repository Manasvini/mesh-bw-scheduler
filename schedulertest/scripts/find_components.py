import json
import sys
with open(sys.argv[1]) as fh:
    data = json.load(fh)

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

all_visited = []
all_nodes = set()
comps = 0
for n in data:
    if n in all_nodes:
        continue
    visited = bfs(data, n)
    comps += 1
    all_visited.append(visited)
    all_nodes = all_nodes.union(set(visited.keys()))

for idx, comp in enumerate(all_visited):
    graph = {}
    for node in comp:
        graph[node] = data[node]
    with open('qmp_' + str(idx) + '.json', 'w') as fh:
        json.dump(graph, fh)
for c in all_visited:
    print(c.keys())
