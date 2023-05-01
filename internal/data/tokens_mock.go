package data

import "time"

type MockTokenModel struct{}

func (m MockTokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	return nil, nil
}

func (m MockTokenModel) Insert(token *Token) error {
	return nil
}

func (m MockTokenModel) DeleteAllForUser(scope string, userID int64) error {
	return nil
}
