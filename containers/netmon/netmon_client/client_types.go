package netmon_client

type IpMapping struct {
	Src	string
	Dst	string
}

type NodeMap struct {
	Mappings []IpMapping
}

