package main

type Bandwidth struct {
	Host  string  `json:"Host"`
	Snd   float64 `json:"Snd"`
	Rcv   float64 `json:"Rcv"`
	SndBw float64 `json:"SndBw"`
	RcvBw float64 `json:"RcvBw"`
}

type Latency struct {
	Host    string
	Latency float64
}
type LatencyResults struct {
	LatencyResults []Latency
}

type Bandwidths []Bandwidth

type BandwidthResults struct {
	BandwidthResults Bandwidths `json:"bandwidthResults"`
}

type TracerouteResults struct {
	TracerouteResults []Traceroute
}

type Traceroute struct {
	Host  string
	Route []string
}
