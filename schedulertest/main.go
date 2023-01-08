package main

import (
    meshscheduler "github.gatech.edu/cs-epl/mesh-bw-scheduler/meshscheduler" 
	gocsv "github.com/gocarina/gocsv"
    "os"
    "flag"
    "fmt"
    "github.com/google/uuid"
	"github.com/golang/glog"
    "time"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: example -stderrthreshold=[INFO|WARNING|FATAL] -log_dir=[string]\n", )
	flag.PrintDefaults()
	os.Exit(2)
}


func readApp(appFilename string, depsFilename string) meshscheduler.Application {
	in, err := os.Open(appFilename)
    if err != nil {
        panic(err)
    }
    defer in.Close()

    app := meshscheduler.Application{}
    componentsMap := make(map[string]meshscheduler.Component, 0)
    components := []*meshscheduler.InputComponent{}
    if err := gocsv.UnmarshalFile(in, &components); err != nil {
        panic(err)
    }

	for _, c := range components {
		comp := meshscheduler.Component{ComponentId:c.Name, Cpu:c.Cpu, Memory:c.Memory}
		componentsMap[c.Name] = comp
	}
	
    in, err = os.Open(depsFilename)
    if err != nil {
        panic(err)
    }
    defer in.Close()


    deps := []*meshscheduler.InputComponentDependency{}
    if err = gocsv.UnmarshalFile(in, &deps); err != nil {
        panic(err)
    }
    
    for _, d :=  range deps {
        srcComp, exists:= componentsMap[d.Src]
        if !exists {
            panic("source for dependency " + d.Src + " not found")
        }
        if len(srcComp.Bandwidth) == 0 {
            srcComp.Bandwidth = make(map[string]int, 0)
        }
        srcComp.Bandwidth[d.Dst] = d.Bandwidth
        componentsMap[d.Src] = srcComp
    }
    app.Components = componentsMap
    for cid, comp := range componentsMap {
        glog.Infof("comp %s has %d deps\n", cid, len(comp.Bandwidth))
    }
    id := uuid.New()
    app.AppId = id.String()
    return app
}

func readNodes(filename string) meshscheduler.NodeMap {
	in, err := os.Open(filename)
    if err != nil {
        panic(err)
    }
    defer in.Close()

    nodes := []*meshscheduler.InputNode{}

    if err := gocsv.UnmarshalFile(in, &nodes); err != nil {
        panic(err)
    }

	nodesMap := make(meshscheduler.NodeMap, 0)
	for _, n := range nodes {
		node := meshscheduler.Node{NodeId:n.NodeId, CpuCapacity:n.Cpu, MemoryCapacity: n.Memory, CpuInUse: 0, MemoryInUse: 0}
		nodesMap[n.NodeId] = node
	}
	return nodesMap
}

func readPaths(filename string, linksMap meshscheduler.LinkMap) meshscheduler.RouteMap {
	in, err := os.Open(filename)
    if err != nil {
        panic(err)
    }
    defer in.Close()

    paths := []*meshscheduler.InputPath{}

    if err := gocsv.UnmarshalFile(in, &paths); err != nil {
        panic(err)
    }

	pathsMap := make(meshscheduler.RouteMap, 0)
	
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
        //fmt.Printf("src %s dst %s p len = %d hop = %s-%s\n", p.Src, p.Dst, len(r.PathBw), r.PathBw[0].Src, r.PathBw[0].Dst)
        pathsMap[p.Src][p.Dst]=r
	
	}
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
						completedCount += 1
					}
					continue
				}
				link, linkExists := linksMap[path.PathBw[hoplen-1].Dst][dst]
                //fmt.Printf("src is %s dst is %s last hop = %s-%s link exists = %d\n", src, dst, linkExists, path.PathBw[hoplen-1].Src, path.PathBw[hoplen-1].Dst)
				if  linkExists {
					path.PathBw = append(path.PathBw, link)
                    pathsMap[src][dst] =path
                } else {
                    //fmt.Printf("check if path from %s to %s is complete\n", path.PathBw[hoplen-1].Dst, dst)
                    _, exists := completedPaths[path.PathBw[hoplen-1].Dst][dst]
                    if exists {
                        completePath := pathsMap[path.PathBw[hoplen-1].Dst][dst].PathBw
                        //fmt.Printf("src = %s dst = %s path from = %s dst = %s is complete\n", src, dst, completePath[0].Dst, completePath[len(completePath)-1].Dst) 
                        path.PathBw = append(path.PathBw, completePath...)
                        pathsMap[src][dst] = path
                    }
                }
			}	
			
		}
	}
    glog.Infof("Finished processing paths\n")
    for src, pathDist := range pathsMap{
        for dst, path:= range pathDist {
            glog.Infof("src = %s dst=%s plen = %d\n", src, dst, len(path.PathBw))
        }
    }
	return pathsMap


}

func readLinks(filename string) meshscheduler.LinkMap {
	in, err := os.Open(filename)
    if err != nil {
        panic(err)
    }
    defer in.Close()

    links := []*meshscheduler.InputLink{}

    if err := gocsv.UnmarshalFile(in, &links); err != nil {
        panic(err)
    }

	linksMap := make(meshscheduler.LinkMap, 0)
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
	defer glog.Flush()

    inputDir := flag.String("i", "./", "input directory containing app and network configs")
    
    flag.Parse()
    opt := meshscheduler.NewOptimalScheduler()
   
    
    links := readLinks(*inputDir + "/links.csv")
    nodes := readNodes(*inputDir + "/nodes.csv")
    paths := readPaths(*inputDir + "/paths.csv", links)
    app := readApp(*inputDir + "/app.csv", *inputDir + "/deps.csv")
    opt.InitScheduler(nodes, paths, links)
    s := time.Now()
    opt.Schedule(app)
    dur := time.Since(s)
    opt.PrintState()
    opt.PrintAssignments()
    fmt.Printf("Scheduling took %.3f ms to execute", float64(dur.Microseconds())/1000.0)
 
}
