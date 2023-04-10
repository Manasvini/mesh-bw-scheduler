import os
import sys
import json
from app import Application
from node import Node, Link, Path
from random import shuffle
class FirstFitK8sScheduler:
    def __init__(self, node_config_file, paths_config_file):
        with open(node_config_file) as fh:
            self.nodeconfig = json.load(fh)
        fh.close()
        print(self.nodeconfig)
        with open(paths_config_file) as fh:
            self.pathsconfig = json.load(fh)
        
        self._init_nodes()

    def _init_nodes(self):
        self.nodes = []
        nodes = {}
        for n in self.nodeconfig['nodes']:
            node = Node(n['node_id'], n['cpu'], n['memory_mb'])
            nodes[n['node_id']] = node
        print('got {} nodes'.format(len(self.nodes))) 

        self.links = {}
        self.paths = {}
        for n in self.nodeconfig['nodes']:
            node = Node(n['node_id'], n['cpu'], n['memory_mb'])
            nodes[n['node_id']] = node

        for l in self.nodeconfig['links']:
            link = Link(l['src'], l['dst'], l['latency_ms'], l['bw'])
            nodes[l['src']].links[l['dst']] = link
            if l['src'] not in self.links:
                self.links[l['src']] ={}
            self.links[l['src']][l['dst']] = link

        for p in self.pathsconfig['paths']:
            if p['src'] not in self.paths:
                self.paths[p['src']] = {}
            plinks = []
            for hop in p['hops']:
                plinks.append(self.links[hop['src']][hop['dst']])
            path = Path(p['src'], p['dst'], plinks)
            path.update_path_bw(0)
            path.set_bottleneck_latency()
            path.set_bottleneck_bw()

            nodes[p['src']].paths[p['dst']] = path
            self.paths[p['src']][p['dst']] = path
    
        for node_id in nodes:
            self.nodes.append(nodes[node_id])
        print('got {} nodes and {} links'.format(len(self.nodes), len(self.nodeconfig['links']))) 
    
    def update_bw_for_deps(self, comp, dst_node, comp_node_assignment, deps):
        print('update bw for ', comp.comp_id, ' node = ', dst_node.node_id)
        for comp_id, src_node_id in comp_node_assignment.items():
            if comp_id not in deps:
                continue
            bw = 0
            if comp.comp_id in deps[comp_id]:
                bw = deps[comp_id][comp.comp_id].bw
            else:
                continue
            src_node = None
            print('bw is ', bw, ' for', comp_id)
            for node in self.nodes:
                print('sle node', node.node_id)
                if src_node_id == node.node_id:
                    src_node = node
                    break
            print('src node', src_node.node_id)
            src_node.update_bw_usage(dst_node.node_id, bw)
            
            
            for src in self.paths:
                for dst in self.paths[src]:
                    p =self.paths[src][dst]
                    p.set_bottleneck_bw()
                    self.paths[src][dst] = p
                    p.print_path()

    def cur_fit(self, cur_comp, cur_node):
        if cur_node.is_cpu_usage_possible(cur_comp.cpu) and \
        cur_node.is_memory_usage_possible(cur_comp.memory):
            return True
        return False

    def get_cluster_state(self):
        return self.nodes

    def schedule(self, app):
        topo_order, possible = app.topo_sort()
        topo_order.sort()
        print(topo_order)
        topo_order.reverse()
        if possible:
            node_idx = -1
            comp_node_assignment = {}
            
            cur_assignments = -1
            prev_assignments = -1
            # k8s bisases for nodes with highest utilization for effective bin packing
            self.nodes.sort()
            self.nodes.reverse()
            cur_comp = None
            in_flight = False
            while len(topo_order) > 0 or in_flight == True:
                print('topo order is', topo_order)
                # We made assignments so resource availability has changed. 
                # Nodes need to be re-sorted
                print(cur_assignments, prev_assignments)
                if cur_assignments != prev_assignments: 
                    self.nodes.sort()
                    self.nodes.reverse()
                    cur_assignments = 0
                    prev_assignments = 0
                else:
                    # We did not make an assignment on the current node 
                    # So we just move to the next node for scheduling this component
                    node_idx += 1
                    cur_assignments = 0
                    prev_assignments = 0
                # we exhausted all possible nodes for this component
                if node_idx == len(self.nodes):
                    return {}, False

                cur_node = self.nodes[node_idx]
                while True:
                    if len(topo_order) == 0 and in_flight == False:
                        break
                    else:
                        if not in_flight:
                            cur_comp_id = topo_order.pop(0)
                            cur_comp = app.comps[cur_comp_id]
                            in_flight = True
                        if self.cur_fit(cur_comp, cur_node):
                            # put as many components on this node as possible
                            # assumption is that within a node
                            comp_node_assignment[cur_comp.comp_id] = cur_node.node_id
                            print('assignment so far', comp_node_assignment)
                            cur_node.update_cpu_usage(cur_comp.cpu)
                            cur_node.update_memory_usage(cur_comp.memory)
              
                            self.update_bw_for_deps(cur_comp, cur_node, comp_node_assignment, app.deps)
              
                            self.nodes[node_idx] = cur_node
                            cur_assignments += 1
                            in_flight = False
                        else:
                            print('cannot fit ', cur_comp_id, ' on node ', cur_node.node_id)
                            in_flight = True
                            break
                #breakpoint()

            return comp_node_assignment, True
        return {}, False


def main():
    sched = FirstFitK8sScheduler('data/qmp_topo1.json', 'data/qmp_topo1_paths.json')
    app = Application('app.json')
    print(sched.schedule(app))

if __name__=='__main__':
    main()

