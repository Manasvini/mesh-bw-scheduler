""" Harvest Containers Cluster """

# Import the Portal object.
import geni.portal as portal
# Import the ProtoGENI library.
import geni.rspec.pg as pg
# Import the Emulab specific extensions.
import geni.rspec.emulab as emulab

# Create a portal object,
pc = portal.Context()

# Create a Request object to start building the RSpec.
request = pc.makeRequestRSpec()


pc.defineParameter( "n", "Number of client machines", portal.ParameterType.INTEGER, 1 )
pc.defineParameter("sameSwitch",  "No Interswitch Links", portal.ParameterType.BOOLEAN, True)
params = pc.bindParameters()
pc.verifyParameters()

# Node server
node_server = request.RawPC('server')
node_server.routable_control_ip = True
node_server.hardware_type = 'c220g2'
#node_server.disk_image = 'urn:publicid:IDN+apt.emulab.net+image+nfslicer-PG0:harvest-k8s'
#node_server.disk_image = 'urn:publicid:IDN+apt.emulab.net+image+nfslicer-PG0:knets-int-test.server'
#node_server.disk_image = 'urn:publicid:IDN+clemson.cloudlab.us+image+nfslicer-PG0:c6420-with-deps'
#node_server.disk_image = 'urn:publicid:IDN+emulab.net+image+emulab-ops//UBUNTU20-64-STD'
# Clean Ubuntu20 install with HarvestContainers repo + custom kernel
node_server.disk_image = 'urn:publicid:IDN+clemson.cloudlab.us+image+nfslicer-PG0:harvest-base'
iface1 = node_server.addInterface()
iface1.addAddress(pg.IPv4Address("192.168.10.10", "255.255.255.0"))
lan = request.LAN("lan")
lan.addInterface(iface1)
#lan.bandwidth = 25000000

# Node client
ifaces = []
for i in range(params.n):
    node_client = request.RawPC('client'+str(i))
    node_client.routable_control_ip = True
    node_client.hardware_type = 'c220g2'
    #node_client.disk_image = 'urn:publicid:IDN+apt.emulab.net+image+nfslicer-PG0:harvest-k8s'
    #node_client.disk_image = 'urn:publicid:IDN+apt.emulab.net+image+nfslicer-PG0:knets-int-test.server'
    #node_client.disk_image = 'urn:publicid:IDN+emulab.net+image+emulab-ops//UBUNTU20-64-STD'
    #node_client.disk_image = 'urn:publicid:IDN+clemson.cloudlab.us+image+nfslicer-PG0:c6420-with-deps'
    # Clean Ubuntu20 install with HarvestContainers repo + custom kernel
    node_client.disk_image = 'urn:publicid:IDN+clemson.cloudlab.us+image+nfslicer-PG0:harvest-base'
    clt_iface = node_client.addInterface()
    clt_iface.addAddress(pg.IPv4Address("192.168.10.1"+str(i+1), "255.255.255.0"))
    ifaces.append(clt_iface)

for iface in ifaces:
    lan.addInterface(iface)


if params.sameSwitch:
    lan.setNoInterSwitchLinks()

# Print the generated rspec
pc.printRequestRSpec(request)
