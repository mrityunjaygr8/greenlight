package data

import (
	"database/sql"
	"errors"
)

var ErrRecordNotFound = errors.New("record not found")
var ErrEditConflict = errors.New("edit conflict")

type Models struct {
	Movies MovieModelInterface
}

func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db},
	}
}

func NewMockModels(db *sql.DB) Models {
	return Models{
		Movies: MockMovieModel{},
	}
}