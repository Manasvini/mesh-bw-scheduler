package main

import (
    meshscheduler "github.gatech.edu/cs-epl/mesh-bw-scheduler/meshscheduler" 
	gocsv "github.com/gocarina/gocsv"
    "os"
    "fmt"
)

func readNodes(filename string) map[string]meshscheduler.Node {
	in, err := os.Open(filename)
    if err != nil {
        panic(err)
    }
    defer in.Close()

    nodes := []*meshscheduler.InputNode{}

    if err := gocsv.UnmarshalFile(in, &nodes); err != nil {
        panic(err)
    }

	nodesMap := make(map[string]meshscheduler.Node, 0)
	for _, n := range nodes {
		node := meshscheduler.Node{NodeId:n.NodeId, CpuCapacity:n.Cpu, MemoryCapacity: n.Memory, CpuInUse: 0, MemoryInUse: 0}
		nodesMap[n.NodeId] = node
	}
	return nodesMap
}

func readPaths(filename string, linksMap map[string]map[string]*meshscheduler.LinkBandwidth) map[string]map[string]meshscheduler.Route {
	in, err := os.Open(filename)
    if err != nil {
        panic(err)
    }
    defer in.Close()

    paths := []*meshscheduler.InputPath{}

    if err := gocsv.UnmarshalFile(in, &paths); err != nil {
        panic(err)
    }

	pathsMap := make(map[string]map[string]meshscheduler.Route, 0)
	
	for src, dstLink := range linksMap {
		for dst, link := range dstLink {
			_, exists := pathsMap[src]
			if !exists {
				pathsMap[src] = make(map[string]meshscheduler.Route, 0)	
			}
			r := meshscheduler.Route{Src:link.Src, Dst:link.Dst}
			r.PathBw = append(r.PathBw, link)
			pathsMap[src][dst] = r
		}
	}
	for _, p := range paths {
		_, exists := pathsMap[p.Src]
		if !exists {
			pathsMap[p.Src] = make(map[string]meshscheduler.Route, 0)	
	    }
        pathsMap[p.Src][p.Dst] = meshscheduler.Route{Src:p.Src, Dst:p.Dst}
		fmt.Printf("add src=%s dst=%s\n", p.Src,p.Dst)
	    r := pathsMap[p.Src][p.Dst]

		if p.NextHop != p.Dst {
		    link, linkExists := linksMap[p.Src][p.NextHop]
		    if linkExists {
			    r.PathBw = append(r.PathBw, link)
		    }
		} else {

            link, linkExists := linksMap[p.Src][p.Dst]
		    if linkExists {
			    r.PathBw = append(r.PathBw, link)
		    }
        }
        pathsMap[p.Src][p.Dst]=r
        fmt.Printf("src= %s dst = %s,  hop path len is %d\n", p.Src, p.Dst, len(pathsMap[p.Src][p.Dst].PathBw))
	
	}
    fmt.Printf("Have %d paths\n", len(paths)) 
	completedCount := 0
	completedPaths := make(map[string]map[string]bool, 0)
	for {
		if completedCount == len(paths) {
			break
		}
        //fmt.Printf("Completed = %d\n", completedCount)
		for src, dstPath := range pathsMap {
			for dst, path := range dstPath {
				hoplen := len(path.PathBw)
                //fmt.Printf("path src=%s dst=%s, last hop=%s\n", path.Src, path.Dst, path.PathBw[hoplen-1].Dst)
				if path.PathBw[hoplen-1].Dst == path.Dst {
					_, exists := completedPaths[src]
                    
					if !exists {
                        completedPaths[src] = make(map[string]bool, 0)
                    }
                    _, exists = completedPaths[src][dst]
                    if !exists{
						completedPaths[src][dst] = true
                        fmt.Printf("add src = %s dst=%s\n", src, dst)
						completedCount += 1
					}
					continue
				}
				link, linkExists := linksMap[path.PathBw[hoplen-1].Dst][dst]
				fmt.Printf("cc=%d src = %s dst = %s hop = %s, exists=%d\n", completedCount, src, dst, path.PathBw[hoplen-1].Dst, linkExists)
				if  linkExists {
					path.PathBw = append(path.PathBw, link)
                    fmt.Printf("Path added %s %s link\n", link.Src, link.Dst)
				
                    pathsMap[src][dst] =path
                }
			}	
			
		}
	}
    fmt.Printf("Finished processing paths\n")
    for src, pathDist := range pathsMap{
        for dst, path:= range pathDist {
            fmt.Printf("src = %s dst=%s plen = %d\n", src, dst, len(path.PathBw))
        }
    }
	return pathsMap


}

func readLinks(filename string) map[string]map[string]*meshscheduler.LinkBandwidth {
	in, err := os.Open(filename)
    if err != nil {
        panic(err)
    }
    defer in.Close()

    links := []*meshscheduler.InputLink{}

    if err := gocsv.UnmarshalFile(in, &links); err != nil {
        panic(err)
    }

	linksMap := make(map[string]map[string]*meshscheduler.LinkBandwidth, 0)
	for _, l := range links {
		link := &meshscheduler.LinkBandwidth{Src:l.Src, Dst:l.Dst, BwCapacity:l.Bw, BwInUse:0}
		_, exists := linksMap[l.Src]
		if !exists {
			linksMap[l.Src] = make(map[string]*meshscheduler.LinkBandwidth, 0)
		}
	
		linksMap[l.Src][l.Dst] = link
	}
	return linksMap

} 

func main() {
    opt := meshscheduler.NewOptimalScheduler()
    
    links := readLinks("links.csv")
    nodes := readNodes("nodes.csv")
    paths := readPaths("paths.csv", links)

    opt.InitScheduler(nodes, paths, links)

}
