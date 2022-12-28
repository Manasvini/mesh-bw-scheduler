from time import time
from copy import deepcopy

from pprint import PrettyPrinter
from itertools import combinations_with_replacement, combinations

from hardcoded_inputs import *
from topo_parse import parse_topo

pp = PrettyPrinter(indent=4)

def get_solutions(topology, application, first=False):
    ret = []

    assignments = 0
    for selection in combinations(topology.keys(), len(application)):
        assignment = dict(zip(application.keys(), selection))
        #print(assignment)
        assignments += 1
        
        topology_ = deepcopy(topology)

        poss = True
        for parent, childset in application.items():
            for child, value in childset.items():
                nodep, nodec = [assignment[comp] for comp in (parent, child)]

                if(parent < child):
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

            if first:
                return ret

    print(f"{assignments} assignments")
    
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


start = time()
out = get_solutions(
    parse_topo("/Users/abauskar/Workspaces/mesh-bw-scheduler/scripts/python_solver/topo/qmp_2022-11-14_09.json"), 
    fill(application),
    True
)
end = time()
print(f"{len(out)} solutions")
pp.pprint(out)
print(f"elapsed {end-start}")
