package data

type MockUsersModel struct{}

func (u MockUsersModel) Insert(user *User) error {
	return nil
}

func (u MockUsersModel) GetByEmail(email string) (*User, error) {
	return nil, nil
}

func (u MockUsersModel) Update(user *User) error {
	return nil
}
