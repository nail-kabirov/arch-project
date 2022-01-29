package streams

const IntegrationEventStreamName = "integration_event"

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
}

type ConfirmationCallback func(id MsgID, confirmed bool, err error)

type Environment interface {
	AddConsumer(streamName string) (Consumer, error)
	AddProducer(streamName string, cb ConfirmationCallback) (Producer, error)
}
