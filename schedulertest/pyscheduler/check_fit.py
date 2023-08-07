import argparse

from app import Application
from k8s_scheduler import FirstFitK8sScheduler 
from mesh_scheduler import FirstFitScheduler


def evaluate_scheduler(scheduler, app):
    placement = scheduler.schedule(app)
    nodes = scheduler.get_cluster_state()

    is_bw_exceeded = {}
    for n in nodes:
        print(n.node_id, n.get_link_total_cap())
        for dst in n.paths:
            n.paths[dst].print_path()
            if n.paths[dst].bw < 0 :
                if n.node_id not in is_bw_exceeded:
                    is_bw_exceeded[n.node_id] = {}
                is_bw_exceeded[n.node_id][dst] = n.paths[dst].bw
    return is_bw_exceeded

def parse_args():
    parser = argparse.ArgumentParser(description='Run schedulers and find fit')
    parser.add_argument('-t','--topofile', help='Topo file', required=True, type=str)
    parser.add_argument('-a','--appfile', help='app file', required=True, type=str)
    parser.add_argument('-p','--pathsfile', help='paths file', required=True, type=str)
    parser.add_argument('-s','--scheduler', help='(k8s/mesh)', required=True, type=str)
  
   
    args = parser.parse_args()
    return args


def main():
    args = parse_args()
    sched = None
    app = Application(args.appfile)
    if args.scheduler == 'k8s':
        sched = FirstFitK8sScheduler(args.topofile, args.pathsfile)
    if args.scheduler == 'mesh':
        sched = FirstFitScheduler(args.topofile, args.pathsfile)

    bw_exceeded = evaluate_scheduler(sched, app)
    print(bw_exceeded)
if __name__=='__main__':
    main()
