package state

type UnlockFunc func()

type Backend interface {
	Load(clusterName string) (*ClusterState, error)
	Save(state *ClusterState) error
	Lock(clusterName string) (UnlockFunc, error)
	List() ([]string, error)
	Delete(clusterName string) error
}
