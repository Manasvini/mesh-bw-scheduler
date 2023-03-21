class Link:
    def __init__(self, src, dst, latency, bw):
        self.src = src
        self.dst = dst
        self.latency = latency
        self.available_bw = bw
        self.used_bw = 0
    
    def is_usage_possible(self, bw):
        if self.used_bw + bw <= self.available_bw:
            return True
        return False
   
    def update_usage(self, bw):
        self.used_bw += bw

    def get_capacity(self):
        return self.available_bw

    def get_usage(self):
        return self.used_bw
     
    def get_latency(self):
        return self.latency

class Path:
    def __init__(self, src, dst, links):
        self.src = src
        self.dst = dst
        self.latency = 0
        self.bw = 0
        self.links = links
        self.set_bottleneck_bw()

    def get_link_available(self, link):
        return link.get_capacity() - link.get_usage()


    def set_bottleneck_bw(self):
        min_bw = self.get_link_available(self.links[0])
        for l in self.links:
            cur_min_bw = self.get_link_available(l)
            if cur_min_bw < min_bw:
                min_bw = cur_min_bw
        self.bw = min_bw

    def update_path_bw(self, bw):
        for i in range(len(self.links)):
            l = self.links[i]
            l.update_usage(bw)
            self.links[i] = l
    
    def set_bottleneck_latency(self):
        max_latency = self.links[0].get_latency()
        for l in self.links:
            cur_max_latency = l.get_latency()
            if cur_max_latency > max_latency:
                max_latency = cur_max_latency
        self.latency = max_latency

    def is_bw_usage_possible(self, bw):
        return bw < self.bw

    def print_path(self):
        print('src=', self.src, 'dst=', self.dst, 'bw = ', self.bw, 'latency = ', self.latency, 'hops ', [ (l.src, l.dst) for l in self.links])

class Node:
    def __init__(self, node_id, cpu, memory):
        self.available_cpu = cpu
        self.available_memory = memory
        self.links = {}
        self.node_id = node_id

        self.used_cpu = 0
        self.used_memory = 0
            
        self.paths = {}

    def add_link(self, dst_id, latency, bw):
        self.links[node_id] = Link(self.node_id, dst_id, latency, bw)

    def add_path(self, dst_id, path):
        self.paths[dst_id] = path

    def is_cpu_usage_possible(self, cpu):
        if self.used_cpu + cpu <= self.available_cpu:
            return True
        return False
    
    def update_cpu_usage(self, cpu):
        self.used_cpu += cpu

    def update_memory_usage(self, memory):
        self.used_memory += memory

    def is_memory_usage_possible(self, memory):
        if self.used_memory + memory <= self.available_memory:
            return True
    
    def get_link_total_cap(self):
        cap = 0
        for dst, link in self.links.items():
            cap += (link.available_bw - link.used_bw)
        return cap

    def get_link_total_available(self):
        cap = 0
        for dst, link in self.links.items():
            cap += link.available_bw
        return cap

    def update_bw_usage(self, node_id, bw):
        if node_id in self.paths:
            path = self.paths[node_id]
            path.update_path_bw(bw)
            path.set_bottleneck_bw()
            self.paths[node_id] = path

    def is_bw_usage_possible(self, node_id, bw):
        if node_id in self.paths:
            return self.paths[node_id].is_bw_usage_possible(bw)
        return False

    def print_usage(self):
        print('total cpu:', self.available_cpu, ' in use cpu:', self.used_cpu)
        print('total mem:', self.available_memory, ' in use mem:', self.used_memory)
        print('total bw:', self.get_link_total_available(), ' in use bw:', self.get_link_total_available() - self.get_link_total_cap())
    
    def _is_memory_lt(self, other):
        this_mem = self.available_memory - self.used_memory
        other_mem = other.available_memory - other.used_memory
        return this_mem >= other_mem

    def _is_cpu_lt(self, other):
        this_cpu = self.available_cpu - self.used_cpu
        other_cpu = other.available_cpu - other.used_cpu
        if this_cpu > other_cpu:
            return True
        if this_cpu == other_cpu:
            return self._is_memory_lt(other)
        return False

    def __lt__(self, other):
        return self._is_cpu_lt(other)
