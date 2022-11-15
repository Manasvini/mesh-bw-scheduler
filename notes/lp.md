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
- A set of services
- A graph of nodes that are communicating with each other
- Every node in graph has independent set of HW requirements.

| Symbol      | Description |
| ----------- | ----------- |
| $K$         | Set of applications. |
| $N_k$       | Number of services in an application $k$. |
| $c_{ki}$    | Compute required for service $i$ of application $k$. |
| $m_{ki}$    | Memory required for service $i$ of application $k$. |
| $s_{ki}$    | Container size for service $i$ of application $k$. |
| $b_{kij}$   | BW required betwee service $i$ and service $j$ of application $k$. |

### Topology Notation

| Symbol      | Description |
| ----------- | ----------- |
| $N$         | Number of nodes in the mesh. |
| $N_a$       | Node $a$ of the mesh |
| $C^{a}$     | Compute present on node $a$. |
| $M^{a}$     | Memory present on node $a$.  |
| $B^{ab}$    | Max BW between on node $a$ and $b$. |

### Assignment Notation

| Symbol           | Description |
| -------------    | ----------- |
| $X^{a}_{kj}(t)$  | equals 1 iff at time $t$, service $j$ of application $k$ is allocated on node $a$ |
| $Y_k(t)$         | equals 1 iff at time $t$, all services of application $k$ are allocated. |
| $E^{ab}_{ki}(t)$ | equals 1 iff from time $t-1$ to $t$, the service $i$ of application $k$ was moved from node $a$ to node $b$. |

$$Y_k(t) = N_k == \sum_{a,j}{} X^{a}_{kj}(t)$$

$$ E^{ab}_{ki}(t) = [X^{a}_{ki}(t-1) - X^{a}_{ki}(t)]X^{b}_{ki}(t) $$

## Modelling

We are going to do two seperate schedulings.
1. Bootstrap scheduling. Just find an intial placement of all applications.
2. Active scheduling. Given the topology to change, what is the minimum downtime transfer to a new placement.

### Objective function

For bootstrap scheduling we just try to maximize the number of applications deployed. So the objective function is simply

$$maximize \ \sum_k Y_k(t)$$

For active scheduling, we try to minimize the transfer time to move nodes from an initial placement $X(t-1)$ to $X(t)$. The objective function in this case would be

$$ minimize \  \sum_k \sum_i \frac{1}{B^{ab}} \sum_a \sum_b E^{ab}_{ki}(t) S_{ki} $$

### Constraints

The following constraints restrict the total HW resources

$$ C^a \geq \sum_k \sum_i c_{ki}X^a_{ki} $$
$$ M^a \geq \sum_k \sum_i m_{ki}X^a_{ki} $$
$$ B^{ab} \geq \sum_k \sum_i \sum_j b_{kij}X^a_{ki}X^b_{kj} $$