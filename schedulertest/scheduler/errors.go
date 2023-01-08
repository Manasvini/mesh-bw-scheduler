package meshscheduler
import ( 
         "fmt"
)

type NotFoundError struct{
    Msg string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("message: %s\n", e.Msg)
}

type InsufficientResourceError struct{
    ResourceType    string
    NodeId          string
}

func (e *InsufficientResourceError) Error() string {
    return fmt.Sprintf("Insufficient resource %s on node %s\n", e.ResourceType, e.NodeId)
}
