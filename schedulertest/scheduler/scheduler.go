package meshscheduler

type Scheduler interface {
    InitScheduler(NodeMap, RouteMap, LinkMap)
    VerifyFit(AppCompAssignment, Application, Component) (bool, error)

    Schedule(Application)
}
