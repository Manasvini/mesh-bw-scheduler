package meshscheduler

type Node struct {
    NodeId              string
    CpuCapacity         int
    CpuInUse            int
    MemoryCapacity      int
    MemoryInUse         int
    BandwidthCapacity   map[string]int  // bw to destination node available
    BandwidthInUse      map[string]int
}

type Component struct {
    ComponentId         string
    Cpu                 int
    Memory              int
    Bandwidth           map[string]int  // bw to other component needed
    TotalBw             int
}

type Application struct {
    AppId               string
    Components          map[string]Component   // component name to requirements
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

func (r *Route) FindBottleneckBw() (int, *LinkBandwidth) {
    if len(r.PathBw) == 0{
        return r.BwCapacity - r.BwInUse, &LinkBandwidth{Src:r.Src, Dst: r.Dst, BwInUse: r.BwInUse, BwCapacity:r.BwCapacity}
    }
    minBw := r.PathBw[0].BwCapacity - r.PathBw[0].BwInUse 
    minBwIdx := 0
    for i := 1; i < len(r.PathBw); i++{
        if r.PathBw[i].BwCapacity - r.PathBw[i].BwInUse < minBw {
            minBw = r.PathBw[i].BwCapacity - r.PathBw[i].BwInUse
            minBwIdx = i
        }
    }
    linkBw := r.PathBw[minBwIdx]
    return minBw, linkBw
}


func (r *Route) SetPathBw(bw int) {
    for i := 0; i < len(r.PathBw); i++ {
        r.PathBw[i].BwInUse = bw
    }
}

func (r *Route) RecomputeBw(bottleneckLink *LinkBandwidth) {
    usesLink := false
    for i := 0; i < len(r.PathBw); i++ {
        if r.PathBw[i].Src == bottleneckLink.Src && r.PathBw[i].Dst == bottleneckLink.Dst {
            usesLink = true
            break
        }
    }
    if usesLink == true{
        bbw, _ := r.FindBottleneckBw()
        r.BwInUse = bbw
    }

}


const DEPLOYED  = "DEPLOYED"
const WAITING   = "WAITING"
const COMPLETED = "COMPLETED"
