module controller_main

go 1.19

replace github.gatech.edu/cs-epl/mesh-bw-scheduler/bwcontroller => /users/msethur1/mesh-bw-scheduler/containers/bw_controller/controller

replace github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon => /users/msethur1/mesh-bw-scheduler/containers/netmon/proto

require (
	github.gatech.edu/cs-epl/mesh-bw-scheduler/bwcontroller v0.0.0-00010101000000-000000000000
	github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon_client v0.0.0-00010101000000-000000000000
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/prometheus/client_golang v1.15.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon v0.0.0-00010101000000-000000000000 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/grpc v1.54.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon_client => /users/msethur1/mesh-bw-scheduler/containers/netmon/netmon_client
