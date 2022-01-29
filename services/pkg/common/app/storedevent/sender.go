package storedevent

import "arch-homework/pkg/common/app/integrationevent"

type Sender interface {
	EventStored(uid integrationevent.EventUID)
	SendStoredEvents()
}
