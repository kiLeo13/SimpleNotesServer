package events

type SocketEvent interface {
	GetType() EventType
}
