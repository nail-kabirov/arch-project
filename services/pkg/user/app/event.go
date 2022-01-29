package app

import (
	"arch-homework/pkg/common/app/integrationevent"
	"arch-homework/pkg/common/app/uuid"
	"encoding/json"
)

const typeUserRegistered = "user.user_registered"

func NewUserRegisteredEvent(userID UserID, login string) integrationevent.EventData {
	body, _ := json.Marshal(userRegisteredEventBody{
		UserID: string(userID),
		Login:  login,
	})

	return integrationevent.EventData{
		UID:  newUID(),
		Type: typeUserRegistered,
		Body: string(body),
	}
}

func newUID() integrationevent.EventUID {
	return integrationevent.EventUID(uuid.GenerateNew())
}

type userRegisteredEventBody struct {
	UserID string `json:"user_id"`
	Login  string `json:"login"`
}
