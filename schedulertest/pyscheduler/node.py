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
      
    
class Node:
    def __init__(self, node_id, cpu, memory):
        self.available_cpu = cpu
        self.available_memory = memory
        self.links = {}
        self.node_id = node_id

        self.used_cpu = 0
        self.used_memory = 0

    def add_link(self, dst_id, latency, bw):
        self.links[node_id] = Link(self.node_id, dst_id, latency, bw)

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
        if node_id in self.links:
            link = self.links[node_id]
            link.used_bw += bw
            self.links[node_id] = link

    def is_bw_usage_possible(self, node_id, bw):
        if node_id in self.links:
            return self.links[node_id].is_usage_possible(bw)
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
