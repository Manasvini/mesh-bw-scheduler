package meshscheduler

type Scheduler interface {
    InitScheduler(map[string]Node, map[string]map[string]Route, map[string]map[string]*LinkBandwidth)
    VerifyFit(map[string]map[string]string, Application, Component) (bool, error)

    Schedule(Application)
}
