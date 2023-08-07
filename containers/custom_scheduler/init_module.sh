PROJECT_ROOT=$1
go mod edit -replace github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon=$PROJECT_ROOT/containers/netmon/proto
go mod edit -replace github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon_client=$PROJECT_ROOT/containers/netmon/netmon_client
