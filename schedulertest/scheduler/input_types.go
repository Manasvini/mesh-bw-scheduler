package meshscheduler

type InputNode struct {
    NodeId      string  `csv:"nodeId"` // .csv column headers
    Cpu         int     `csv:"cpu"`
    Memory      int     `csv:"memory_mb"`
}


type InputLink struct {
    Src         string  `csv:"src"`
    Dst         string  `csv:"dst"`
    Bw          int     `csv:"bw_mb"`
}

type InputPath struct {
    Src         string  `csv:"src"`
    Dst         string  `csv:"dst"`
    NextHop     string  `csv:"next_hop"`
}
