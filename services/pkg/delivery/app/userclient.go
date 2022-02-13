package app

type UserInfo struct {
	Login     string
	FirstName string
	LastName  string
	Address   string
}

type UserServiceClient interface {
	GetUserInfo(userID UserID) (UserInfo, error)
}
