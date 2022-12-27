class UnionFind:
    p = {}
    w = {}

    def __init__(self, nodes):
        for node in nodes:
            self.p[node] = node
            self.w[node] = 1

    def get_root(self, node):
        while node != self.p[node]:
            p1 = self.p[node]
            p2 = self.p[p1]

            self.w[p1] -= self.w[node]
            self.w[p2] += self.w[node]
            self.p[node] = p2
            node = p2

        return node

    def is_same(self, node1, node2):
        return self.get_root(node1) == self.get_root(node2)

    def join(self, node1, node2):
        if self.is_same(node1, node2):
            return False

        p1, p2 = [self.get_root(node) for node in [node1, node2]]

        if self.w[p1] > self.w[p2]:
            self.p[p2] = p1
            self.w[p1] += self.w[p2]
        else:
            self.p[p1] = p2
            self.w[p2] += self.w[p1]

        return True
        