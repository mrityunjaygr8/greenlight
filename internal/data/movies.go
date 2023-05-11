package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/mrityunjaygr8/greenlight/dbmodels"
	"github.com/mrityunjaygr8/greenlight/internal/validator"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"github.com/volatiletech/sqlboiler/v4/types"
)

type Movie struct {
	ID        int64             `json:"id"`
	CreatedAt time.Time         `json:"-"`
	Title     string            `json:"title"`
	Year      int32             `json:"year,omitempty"`
	Runtime   Runtime           `json:"runtime,omitempty"`
	Genres    types.StringArray `json:"genres,omitempty"`
	Version   int32             `json:"version"`
}

type MovieModelInterface interface {
	GetAll(title string, genres []string, filters Filters) (*[]Movie, Metadata, error)
	Insert(movie *Movie) error
	Get(id int64) (*Movie, error)
	Update(movie *Movie) error
	Delete(id int64) error
}

type MovieModel struct {
	DB *sql.DB
}

func dbModelToMovie(dbMovie dbmodels.Movie) *Movie {
	movie := &Movie{
		ID:        dbMovie.ID,
		CreatedAt: dbMovie.CreatedAt,
		Title:     dbMovie.Title,
		Year:      int32(dbMovie.Year),
		Runtime:   Runtime(dbMovie.Runtime),
		Genres:    dbMovie.Genres,
		Version:   int32(dbMovie.Version),
	}

	return movie
}

func movieToDbModel(movie Movie) *dbmodels.Movie {
	dbMovie := &dbmodels.Movie{
		ID:        movie.ID,
		CreatedAt: movie.CreatedAt,
		Title:     movie.Title,
		Year:      int(movie.Year),
		Runtime:   int(movie.Runtime),
		Genres:    movie.Genres,
		Version:   int(movie.Version),
	}

	return dbMovie
}

func (m MovieModel) GetAll(title string, genres []string, filters Filters) (*[]Movie, Metadata, error) {
	query := make([]qm.QueryMod, 0)

	if len(genres) > 0 {
		query = append(query, qm.Where("genres @> ?", types.Array(genres)))
	}

	if title != "" {
		query = append(query, qm.Where("to_tsvector('simple', title) @@ plainto_tsquery('simple', ?)", title))
	}
	ctx_1, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	totalRecords, err := dbmodels.Movies(query...).Count(ctx_1, m.DB)
	if err != nil {
		return nil, Metadata{}, err
	}

	ctx_2, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	queryOrdering := []qm.QueryMod{
		qm.Limit(filters.limit()),
		qm.Offset(filters.offset()),
		qm.OrderBy(dbmodels.MovieColumns.ID),
	}
	queriesWithOrder := append(query, queryOrdering...)

	moviesSQL := make([]Movie, 0)
	err = dbmodels.Movies(queriesWithOrder...).Bind(ctx_2, m.DB, &moviesSQL)
	if err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(int(totalRecords), filters.Page, filters.PageSize)

	return &moviesSQL, metadata, nil
}

func (m MovieModel) Insert(movie *Movie) error {
	movieDB := movieToDbModel(*movie)
	movieDB.Version = 1

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	err = movieDB.Insert(ctx, tx, boil.Infer())
	if err != nil {
		txErr := tx.Rollback()
		if txErr != nil {
			return txErr
		}

		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	movie.ID = movieDB.ID
	movie.Version = int32(movieDB.Version)

	return nil
}

func (m MovieModel) Get(id int64) (*Movie, error) {
	var movie Movie

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	err := dbmodels.Movies(dbmodels.MovieWhere.ID.EQ(id)).Bind(ctx, m.DB, &movie)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &movie, nil
}

func (m MovieModel) Update(movie *Movie) error {
	dbMovie := movieToDbModel(*movie)
	dbMovie.Version = dbMovie.Version + 1

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	rowsAff, err := dbMovie.Update(ctx, tx, boil.Infer())

	if rowsAff != 1 {
		txErr := tx.Rollback()
		if err != nil {
			return txErr
		}

		return ErrRecordNotFound
	}

	if err != nil {
		txErr := tx.Rollback()
		if err != nil {
			return txErr
		}
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	*movie = *dbModelToMovie(*dbMovie)
	return nil
}

func (m MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	rowsAffected, err := dbmodels.Movies(dbmodels.MovieWhere.ID.EQ(id)).DeleteAll(ctx, tx)

	if rowsAffected != 1 {
		txErr := tx.Rollback()
		if txErr != nil {
			return txErr
		}

		return ErrRecordNotFound
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")

	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}
