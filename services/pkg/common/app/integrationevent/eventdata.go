package integrationevent

import "arch-homework/pkg/common/app/uuid"

type EventUID uuid.UUID

type EventData struct {
	UID  EventUID
	Type string
	Body string
}
