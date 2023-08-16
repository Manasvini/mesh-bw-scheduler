package main

type Bandwidth struct {
	Host string
	Snd  float64
	Rcv  float64
	SndBw int64
	RcvBw int64
}

type BandwidthResults struct {
	BandwidthResults []Bandwidth
}
type TracerouteResults struct {
	TracerouteResults []Traceroute
}

type Traceroute struct {
	Host  string
	Route []string
}
