from input import get_input
from itertools import combinations_with_replacement, combinations
from unionfind import UnionFind

def get_solutions(topology, application):
    ret = []

    for selection in combinations(topology.keys(), len(application)):
        assignment = dict(zip(application.keys(), selection))
        
        topology_ = topology

        poss = True
        for parent, childset in application.items():
            for child, value in childset.items():
                nodep, nodec = [assignment[comp] for comp in (parent, child)]

                if(nodep < nodec):
                    continue

                path = get_path_with_min_weight(topology_, nodep, nodec, value)

                if len(path) == 0:
                    poss = False
                    break
                elif len(path) == 1:
                    continue

                for i in range(0, len(path) - 1):
                    topology_[path[i]][path[i+1]] -= value
                    topology_[path[i+1]][path[i]] -= value

        if poss:
            ret.append(assignment)
    
    return ret

def get_path_with_min_weight(topology, n1, n2, w):
    if n1 == n2:
        return [n1]

    frontier = []
    p = {}

    frontier.append(n1)
    p[n1] = n1

    while len(frontier) != 0:
        curr = frontier.pop()

        for next, value in topology[curr].items():
            if (next in p) or (value < w):
                continue

            frontier.append(next)
            p[next] = curr
        
    if n2 not in p:
        return []
    
    ret = []
    while p[n2] != n2:
        ret.append(n2)
        n2 = p[n2]
    ret.append(n2)
    return ret

print(get_solutions(get_input()[0], get_input()[1]))