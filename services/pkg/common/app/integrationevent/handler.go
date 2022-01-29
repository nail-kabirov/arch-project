package integrationevent

type EventHandler interface {
	Handle(event EventData) error
}
