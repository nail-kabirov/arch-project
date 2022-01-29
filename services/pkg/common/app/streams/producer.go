package streams

type MsgID int64

type Msg struct {
	ID   MsgID
	Body string
}

type Producer interface {
	Send(msg Msg) error
	BatchSend(messages []Msg) error
}
