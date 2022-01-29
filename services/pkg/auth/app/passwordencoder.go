package app

type PasswordEncoder interface {
	Encode(rawPassword string, userID UserID) Password
}
