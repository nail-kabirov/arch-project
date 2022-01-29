package encoding

import (
	"crypto/sha256"
	"fmt"

	"arch-homework/pkg/auth/app"
)

func NewPasswordEncoder() app.PasswordEncoder {
	return sha256PasswordEncoder{}
}

type sha256PasswordEncoder struct {
}

func (m sha256PasswordEncoder) Encode(rawPassword string, userID app.UserID) app.Password {
	data := []byte(string(userID) + rawPassword)
	return app.Password(fmt.Sprintf("%x", sha256.Sum256(data)))
}
