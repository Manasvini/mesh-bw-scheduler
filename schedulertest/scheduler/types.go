package meshscheduler
import ("github.com/golang/glog"
        )
type Node struct {
    NodeId              string
    CpuCapacity         int
    CpuInUse            int
    MemoryCapacity      int
    MemoryInUse         int
}

type ComponentBw        map[string]int                  // bw to other component needed
type ComponentMap       map[string]Component            // component name -> component map

type Component struct {
    ComponentId         string
    Cpu                 int
    Memory              int
    Bandwidth           ComponentBw
    TotalBw             int
}

type Application struct {
    AppId               string
    Components          ComponentMap   
}


type LinkBandwidth struct {
    Src                 string
    Dst                 string
    BwCapacity          int
    BwInUse             int
}

type Route struct {
    Src                 string
    Dst                 string
    BwCapacity          int
    BwInUse             int
    PathBw              []*LinkBandwidth
}

type NodeMap            map[string]Node                         // node id -> node map
type RouteMap           map[string]map[string]Route             // src-> dst -> Route 
type AppCompAssignment  map[string]map[string]string            // app id -> component id -> node id
type LinkMap            map[string]map[string]*LinkBandwidth    // src -> dst -> Link
type DeploymentStateMap map[string]string                       // app -> deployment status

func (r *Route) FindBottleneckBw() (int, *LinkBandwidth) {
    minBw := r.PathBw[0].BwCapacity - r.PathBw[0].BwInUse 
    minBwIdx := 0
    for i := 1; i < len(r.PathBw); i++{
        if r.PathBw[i].BwCapacity - r.PathBw[i].BwInUse < minBw {
            minBw = r.PathBw[i].BwCapacity - r.PathBw[i].BwInUse
            minBwIdx = i
        }
    }
    //glog.Infof("min = %d at idx %d src=%s dst=%s\n", minBw, minBwIdx, r.PathBw[minBwIdx].Src, r.PathBw[minBwIdx].Dst)    
    linkBw := r.PathBw[minBwIdx]
    return minBw, linkBw
}


func (r *Route) SetPathBw(bw int) {
    r.BwInUse = bw
    for i := 0; i < len(r.PathBw); i++ {
        r.PathBw[i].BwInUse = bw
    }
}

func (r *Route) RecomputeBw(bottleneckLink *LinkBandwidth) {
    usesLink := false
    for i := 0; i < len(r.PathBw); i++ {
        if (r.PathBw[i].Src == bottleneckLink.Src && r.PathBw[i].Dst == bottleneckLink.Dst)  {
            usesLink = true
            break
        }
    }
    if usesLink == true{
        r.BwInUse =bottleneckLink.BwInUse
        glog.Infof("Updated src %s dst %s bw to %d\n", r.Src, r.Dst, r.BwInUse)
    }

}


const DEPLOYED  = "DEPLOYED"
const WAITING   = "WAITING"
const COMPLETED = "COMPLETED"
