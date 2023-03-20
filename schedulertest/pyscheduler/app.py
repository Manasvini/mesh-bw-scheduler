import os
import json

class Dependency:
    def __init__(self, src, dst, latency, bw):
        self.src = src
        self.dst = dst
        self.latency = latency
        self.bw = bw

class Component:
    def __init__(self, comp_id, cpu, memory):
        self.cpu = cpu
        self.memory = memory
        self.comp_id = comp_id
        self.deps = {}

    def add_dep(self, dst_comp, latency, bw):
        self.deps[dst_comp] = Dependency(self.comp_id, dst_comp, latency, bw)

class Application:
    def _init_requirements(self, spec):
        for comp in spec['components']:
            self.comps[comp['comp_id']] = Component(comp['comp_id'], comp['cpu'], comp['memory_mb'])
            self.deps[comp['comp_id']] = {}
        for dep in spec['dependencies']:
            self.deps[dep['src']][dep['dst']] = Dependency(dep['src'], dep['dst'], dep['latency_ms'], dep['bw'])
        print("Got app with {} components and {} deps".format(len(spec['components']), len(spec['dependencies'])))
    def __init__(self, spec_file):
        with open(spec_file) as fh:
            spec = json.load(fh)
        self.spec = spec
        self.comps = {}
        self.deps = {}
        self._init_requirements(spec)
   
    def _init_graph(self):
        g = {comp :{} for comp in self.comps}
        for src_comp_id in self.deps:
            for dst_comp_id in self.deps[src_comp_id]:
                g[dst_comp_id][src_comp_id] = 1
        print(g)
        return g

    def _compute_indegrees(self, graph):
        indegrees = {}
        for src in graph:
            indegree = 0
            for dst in graph:
                 if src in graph[dst] and graph[dst][src] == 1:
                    indegree += 1
            indegrees[src] = indegree
        return indegrees

    def _find_zero_indegree_node(self, indegrees):
        nodes = []
        for src in indegrees:
            if indegrees[src] == 0:
                nodes.append(src)
        return nodes

    def topo_sort(self):
        sort_order = []
        g = self._init_graph()
        indegrees = self._compute_indegrees(g)
        zero_indegree_nodes = self._find_zero_indegree_node(indegrees)
        print(zero_indegree_nodes)
        while len(zero_indegree_nodes) > 0:
            curnode = zero_indegree_nodes.pop(0)
            sort_order.append(curnode)
            for dst in g[curnode]:
                g[curnode][dst] = 0
                dst_indegree = 0
                for src in g:
                    if dst in g[src] and g[src][dst] == 1:
                        dst_indegree += 1
                if dst_indegree == 0 and dst not in zero_indegree_nodes:
                    zero_indegree_nodes.append(dst)
    
        edges = 0
        for src in g:
            for dst in g[src]:
                if g[src][dst] == 1:
                    return [], None
                
        return sort_order, True


def main():
   app = Application('app.json')
   print(app.topo_sort())

if __name__=='__main__':
    main()
