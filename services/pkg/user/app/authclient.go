package app

type AuthServiceClient interface {
	RegisterUser(login, password string) (UserID, error)
	RemoveUser(userID UserID) error
}
