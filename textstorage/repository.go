package textstorage

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"
)

// Repository Main Object.
type Repository struct {
	db *goqu.Database
}

// New constructor.
func New(
	db *goqu.Database,
) *Repository {
	return &Repository{
		db: db,
	}
}

func (s *Repository) Text(ctx context.Context, id int64) (string, error) {
	sqlSelect := s.db.From("textstorage_text").
		Select("text").
		Where(goqu.I("id").Eq(id))

	result := ""
	success, err := sqlSelect.Executor().ScanValContext(ctx, &result)
	if err != nil {
		return "", err
	}

	if !success {
		return "", fmt.Errorf("text `%v` not found", id)
	}

	return result, nil
}
