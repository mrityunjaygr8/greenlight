package data

type MockPermissionsModel struct{}

func (p MockPermissionsModel) GetAllForUser(userID int64) (Permissions, error) {
	return nil, nil
}

func (p MockPermissionsModel) AddForUser(userID int64, codes ...string) error {
	return nil
}
