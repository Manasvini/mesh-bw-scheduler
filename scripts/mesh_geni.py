"""A fully connected mesh using multiplexed best effort links. Since most nodes have a small number of
physical interfaces, we must set the links to use vlan ecapsulation over the physical link. On your
nodes, each one of the links will be implemented using a vlan network device. 

Instructions:
Log into your node, use `sudo` to poke around.
"""

# Import the Portal object.
import geni.portal as portal
# Import the ProtoGENI library.
import geni.rspec.pg as pg
# Import the Emulab specific extensions.
import geni.rspec.emulab as emulab


# Array of nodes
nodes = {}

# Create a portal context.
pc = portal.Context()

# Create a Request object to start building the RSpec.
request = pc.makeRequestRSpec()

# Create all the router nodes.
for i in range(0, 5):
    node = request.XenVM("node%d" % (i ))
    node.cores = 1
    node.ram = 2048
    
    node.disk_image = 'urn:publicid:IDN+emulab.net+image+emulab-ops//UBUNTU18-64-STD'
    #nodes.append(node)
    nodes["node%d" % (i )] = node
    pass

# Create the compute nodes
for i in range(0, 5):
    node = request.XenVM("node%d" % (i + 5 ))
    
    if i % 2 == 0:
        node.cores = 8

        # Request a specific amount of memory (in GB).
        node.ram = 8192
    else:
        node.cores = 4
        node.ram = 4096
    node.disk_image = 'urn:publicid:IDN+emulab.net+image+Westside:MeshTestAug17.server:0'
    #nodes.append(node)
    nodes["node%d" % (i + 5 )] = node
    pass

topo = {
  "links":[
         {"src":"node0", "dst":"node1" },
         {"src":"node0", "dst":"node2" },
         {"src":"node0", "dst":"node4" },
         {"src":"node1", "dst":"node2" },
         {"src":"node2", "dst":"node3" },
         {"src":"node3", "dst":"node4" },
         {"src":"node0", "dst":"node5" },
         {"src":"node1", "dst":"node6" },
         {"src":"node2", "dst":"node7" },
         {"src":"node3", "dst":"node8" },
         {"src":"node4", "dst":"node9" },
     ]
}


for link in topo['links']:
    nodeA = nodes[link['src']]
    nodeB = nodes[link['dst']]
    link = request.Link(members = [nodeA, nodeB])
    pass

pc.printRequestRSpec(request)
