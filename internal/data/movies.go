package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	"github.com/mrityunjaygr8/greenlight/dbmodels"
	"github.com/mrityunjaygr8/greenlight/internal/validator"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"github.com/volatiletech/sqlboiler/v4/types"
)

type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   Runtime   `json:"runtime,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}

type MovieModelInterface interface {
	GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error)
	Insert(movie *Movie) error
	Get(id int64) (*Movie, error)
	Update(movie *Movie) error
	Delete(id int64) error
}

type MovieModel struct {
	DB *sql.DB
}

func dbToMovieModel(m dbmodels.Movie) (*Movie, error) {
	movie := &Movie{
		ID:        m.ID,
		CreatedAt: m.CreatedAt,
		Year:      int32(m.Year),
		Title:     m.Title,
		Genres:    []string(m.Genres),
		Runtime:   Runtime(m.Runtime),
		Version:   int32(m.Version),
	}

	return movie, nil
}

func dbToMovieModelSlice(movieSlice dbmodels.MovieSlice) ([]*Movie, error) {
	movies := make([]*Movie, 0)
	for _, mov := range movieSlice {
		movie, err := dbToMovieModel(*mov)
		if err != nil {
			return []*Movie{}, err
		}

		movies = append(movies, movie)
	}

	return movies, nil
}

func (m MovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error) {
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
	moviesSQL, err := dbmodels.Movies(queriesWithOrder...).All(ctx_2, m.DB)
	if err != nil {
		return nil, Metadata{}, err
	}

	movies, err := dbToMovieModelSlice(moviesSQL)
	if err != nil {
		return []*Movie{}, Metadata{}, err
	}

	metadata := calculateMetadata(int(totalRecords), filters.Page, filters.PageSize)

	return movies, metadata, nil
}

func (m MovieModel) Insert(movie *Movie) error {
	query := `
INSERT INTO movies (title, year, runtime, genres)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at, version`

	args := []interface{}{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

func (m MovieModel) Get(id int64) (*Movie, error) {
	query := `SELECT  id, created_at, title, year, runtime, genres, version
FROM movies
WHERE id = $1`

	var movie Movie

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(&movie.ID, &movie.CreatedAt, &movie.Title, &movie.Year, &movie.Runtime, pq.Array(&movie.Genres), &movie.Version)

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
	query := `UPDATE movies
SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
WHERE id = $5 AND version =$6
RETURNING version`

	args := []interface{}{
		movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
		movie.ID,
		movie.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.Version)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}
	query := `DELETE FROM movies WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
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
