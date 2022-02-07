package app

type UserClient interface {
	GetUserLogin(userID UserID) (string, error)
}
