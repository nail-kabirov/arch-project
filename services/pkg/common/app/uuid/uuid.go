package uuid

import (
	gouuid "github.com/satori/go.uuid"
)

type UUID string

func GenerateNew() UUID {
	return UUID(gouuid.NewV4().String())
}

func ValidateUUID(s string) error {
	_, err := gouuid.FromString(s)
	return err
}
