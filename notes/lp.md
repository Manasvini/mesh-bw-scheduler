### Scheduler interface

- When a client connects to a node with a new set of applications to use.
  - Node + {Applications} (?incremental)
- When a new application is setup on a set of nodes.
  - {Nodes} + Application
- Topology changes

Scheduler outputs a placement that satisfies the following

- single node constraints
  - Node compute and memory requirements
- All pairwise node constraints
  - Inter node bw requirements

## Notation



### Application Requirement Notation

An application is
- A set of components.
- A graph of nodes that are communicating with each other
- Every node in graph has independent set of HW requirements.

| Symbol      | Description |
| ----------- | ----------- |
| $A$         | Set of applications. |
| $A_k$       | Application $k$. |
| $\|A_k\|$   | Number of components in Application $k$. |
| $p_{ki}$    | Compute required for component $i$ of application $k$. |
| $m_{ki}$    | Memory required for component $i$ of application $k$. |
| $s_{ki}$    | Container size for component $i$ of application $k$. |
| $b_{kij}$   | BW required between component $i$ and component $j$ of application $k$. |
| $\hat{b}_{ki}$    | BW required for a client to communicate with component $i$ of application $k$.  |
| $l_{kij}$   | Latency required between component $i$ and component $j$ of application $k$. |
| $\hat{l}_{ki}$    | Latency required for a client to communicate with component $i$ of application $k$.  |

### Topology Notation

| Symbol      | Description |
| ----------- | ----------- |
| $T$         | Topology |
| $\|T\|$     | Number of nodes in the topology. |
| $T^{a}$     | Node $a$ of the topology |
| $P^{a}$     | Compute present on node $a$ |
| $M^{a}$     | Memory present on node $a$  |
| $B^{ab}$    | Avg BW between node $a$ and $b$ |
| $L^{ab}$    | Avg latency between node $a$ and $b$ |

### Client Notation

| Symbol      | Description |
| ----------- | ----------- |
| $C$         | Clients in the system |
| $C_l$       | Client $l$ |
| $\hat{B}^{a}_{l}$| Max bandwidth between client $l$ to node $a$ | 
| $C^{a}_{lk}$| equals 1 iff client $l$ connects to application $k$ at node $a$ | 

### Assignment Notation

| Symbol           | Description |
| -------------    | ----------- |
| $X^{a}_{kj}(t)$  | equals 1 iff at time $t$, service $j$ of application $k$ is allocated on node $a$ |
| $Y_k(t)$         | equals 1 iff at time $t$, all services of application $k$ are allocated. |
| $E^{ab}_{ki}(t)$ | equals 1 iff from time $t-1$ to $t$, the service $i$ of application $k$ was moved from node $a$ to node $b$. |

$$Y_k(t) = [ N_k ==  \sum_{a,j}{} X^{a}_{kj}(t) ]$$

$$ E^{ab}_{ki}(t) = [X^{a}_{ki}(t-1) - X^{a}_{ki}(t)]X^{b}_{ki}(t) $$

### Example

Time = 1

<img src="images/img.png" width="300"/>

$$ X^{4}_{1,1}(1) = X^{5}_{1,2}(1) = 1 $$
$$ Y_k(1) = 2 $$

$$ C^{8}_{1,1} = C^{8}_{2,1} = C^{9}_{3,1} = C^{3}_{4,1} = C^{7}_{5,1} = 1 $$

Time = 2

<img src="images/img2.png" width="300"/>

$$ E^{5,6}_{1,2}(2) = 1 $$


## Modelling

We are going to do three seperate schedulings.
1. Application time scheduling
2. Client time scheduling
3. Topology scheduling (includes latency measurements)

### Constraints

The following constraints restrict the total HW resources

$$ P^a \geq \sum_{k,i} p_{ki}X^a_{ki} $$
$$ M^a \geq \sum_{k,i} m_{ki}X^a_{ki} $$
$$ B^{ab} \geq \sum_{k,i,j} b_{kij}X^a_{ki}X^b_{kj} + \sum_{l,k,i} \hat{b}_{ki} X^a_{ki} C^b_{lk}  $$
$$ \hat{B}^{a}_{l} \geq \sum_{k,i} \hat{b}_{ki} C^{a}_{lk} $$
$$ L^{ab} \leq l_{kij} X^a_{ki} X^b_{kj}  $$
$$ L^{ab} \leq \hat{l}_{ki} C^{a}_{lk} X^b_{ki} $$

### Objective function

For bootstrap scheduling we just try to maximize the number of applications deployed. So the objective function is simply

$$maximize \ \sum_k Y_k(t)$$

For active scheduling, we try to minimize the transfer time to move nodes from an initial placement $X(t-1)$ to $X(t)$. The objective function in this case would be

$$ minimize \   ( maximum_{a,b,k}  \ \sum_{i} \frac{1}{B^{ab}}  E^{ab}_{ki}(t) s_{ki} ) $$

## Example

### Video calls

<img src="images/app-video.drawio.png" width="100"/>

1. Centralized video calling app
   1. Unforgiving. High bandwidth + low latency.
   2. Need 2MBps from client to server with less than 100ms latency.
   3. Need 8vcpus and 16GB memory to load and process video, audio streams.

### IOT - Security System

<img src="images/app-iot.drawio.png" width="300"/>

1. Realtime data collections. Every client is a camera.
   1. High bandwidth, low latency.
2. Data server.
   1. Collects data streams from camera and preprocesses it.
   2. Needs lot of CPU to process camera streams and generate metrics. Number of faces? Actions? etc.
3. Control server.
   1. Takes metrics collected from data server and processes them to generate actions in the system.
   2. Lock doors, Increase data quality? Enable other sensors? etc
   3. Low bandwidth, low latency with the clients. High bandwidth, low latency with the data server.

### Social media

<img src="images/app-social media.drawio.png" width="500"/>

1. Auth
   1. Forgiving. Low bandwidth + high latency between client-server, server-server.
   2. 256kbps, 10ms.
2. Set of chat related services.
   1. Chats through this server. low bandwidth + low latency from clients. 
   2. Talks with auth service to refresh sessions to the chat server. Low bandwidth + low latency.
   3. Stores chats locally (no external DB needed).
3. Posting + timeline + Friends services
   1. All posts are made into this server. High bandwidth + high latency from clients.
   2. Timeline requests. High bandwidth + high latency.
   3. Client-client. High bandwidth, low latency.
   4. Needs some cpu and memory but not so much. 
   5. Talks with external DB.
4. DB.
   1. Stores posts, timelines, chats etc.
   2. Only talks with (3). High bandwidth, low latency. (20MBPS, 10ms).
   3. Needs cpu + Memory + storage. 12vcpus, 16GB, 1TB storage.

## Complexity Modelling

1. Routing tables
   1. This information can be assumed to be passed to central scheduler.
   2. Scheduler can then tie up the entire graph and routing paths.
   3. So the scheduler is aware of the system.
   4. **The scheduler does not change the routing tables (we do not control the network layer).**
   5. But we assume that the routing is fixed otherwise we migrate.
   6. **Can we have application specific routing tables by using existing network protocols?**
   7. For theoretical modelling we can safely assume that we know the graph.
2. Simplification 1 (S1):
   1. Assume that path are not considered as well as components are 1 to 1 map.
   2. Problem: Find a graph isomorphism (C -> T) ... (where the weights in T are upper bounded by the mapped weights in C).
   3. Reduce graph isomorphism to this prob. (trivial, weight = 1)
   4. Reverse (unecessary) : Take a weighted C, T. For every edge, replace it with V edges where V is the value of the edge.
3. Simplification 2 (S2):
   1. Assume that only components are 1 to 1 mapped.
   2. Now we can convert the input topology to a complete graph in O(n^2) time. 
   3. So let us consider the input to the problem as a complete graph. 
   4. Problem: Find a graph isomorphism (C -> K) where the weights in T are K are upper bounded by mapped weights in C and K is a complete graph.
   5. Generate complete graph using the path bw and then on this use the clique.