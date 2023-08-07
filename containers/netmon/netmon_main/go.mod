module netmon_main

go 1.19

replace github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon => "/users/msethu0/mesh-bw-scheduler//containers/netmon/proto"

require (
	github.com/iovisor/gobpf v0.2.0
	github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.54.0
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/protobuf v1.30.0 // indirect

	github.com/iovisor/gobpf v0.2.1-0.20221005153822-16120a1bf4d4
	)