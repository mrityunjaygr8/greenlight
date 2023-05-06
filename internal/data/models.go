package data

import (
	"database/sql"
	"errors"
)

var ErrRecordNotFound = errors.New("record not found")
var ErrEditConflict = errors.New("edit conflict")

type Models struct {
	Movies      MovieModelInterface
	Users       UserModelInterface
	Tokens      TokenModelInterface
	Permissions PermissionModelInterface
}

func NewModels(db *sql.DB) Models {
	return Models{
		Movies:      MovieModel{DB: db},
		Users:       UserModel{DB: db},
		Tokens:      TokenModel{DB: db},
		Permissions: PermissionModel{DB: db},
	}
}

func NewMockModels(db *sql.DB) Models {
	return Models{
		Movies:      MockMovieModel{},
		Users:       MockUsersModel{},
		Tokens:      MockTokenModel{},
		Permissions: MockPermissionsModel{},
	}
}
