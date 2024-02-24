package nodemanager

type ClientManagerInterface interface {
	GetBalance(address string) (string, error)
	GetNodeName() string
	IsReady() bool
}
