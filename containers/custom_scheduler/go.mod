module custom_scheduler

go 1.19

require github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon_client v0.0.0-00010101000000-000000000000

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon v0.0.0-00010101000000-000000000000 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/grpc v1.54.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/apimachinery v0.27.1 // indirect
)

replace github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon_client => /home/cvuser/mesh-bw-scheduler/containers/netmon/netmon_client

replace github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon => /home/cvuser/mesh-bw-scheduler/containers/netmon/proto
