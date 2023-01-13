package meshscheduler

type InputNode struct {
	NodeId string `csv:"nodeId"` // .csv column headers
	Cpu    int    `csv:"cpu"`
	Memory int    `csv:"memory_mb"`
}

type InputLink struct {
	Src string `csv:"src"`
	Dst string `csv:"dst"`
	Bw  float64    `csv:"bw_mbps"`
}

type InputPath struct {
	Src     string `csv:"src"`
	Dst     string `csv:"dst"`
	NextHop string `csv:"next_hop"`
}

type InputComponent struct {
	Name   string `csv:"name"`
	Cpu    int    `csv:"cpu"`
	Memory int    `csv:"memory"`
}

type InputComponentDependency struct {
	Src       string `csv:"src"`
	Dst       string `csv:"dst"`
	Bandwidth float64    `csv:"bw_mbps"`
}
