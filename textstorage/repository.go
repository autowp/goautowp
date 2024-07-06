package textstorage

import (
	"context"
	"errors"
	"fmt"

	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

var errTextNotFound = errors.New("text not found")

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
	sqlSelect := s.db.From(schema.TextstorageTextTable).
		Select(schema.TextstorageTextTableTextCol).
		Where(schema.TextstorageTextTableIDCol.Eq(id))

	result := ""

	success, err := sqlSelect.Executor().ScanValContext(ctx, &result)
	if err != nil {
		return "", err
	}

	if !success {
		return "", fmt.Errorf("%w: `%v`", errTextNotFound, id)
	}

	return result, nil
}

func (s *Repository) FirstText(ctx context.Context, ids []int64) (string, error) {
	if len(ids) == 0 {
		return "", nil
	}

	args := append([]interface{}{schema.TextstorageTextTableIDColName}, ids)
	result := ""

	success, err := s.db.From(schema.TextstorageTextTable).
		Select(schema.TextstorageTextTableTextCol).
		Where(
			schema.TextstorageTextTableIDCol.In(ids),
			goqu.Func("length", schema.TextstorageTextTableTextCol).Gt(0),
		).
		Order(goqu.Func("field", args...).Asc()).
		Limit(1).
		ScanValContext(ctx, &result)
	if err != nil {
		return "", err
	}

	if !success {
		return "", fmt.Errorf("%w: `%v`", errTextNotFound, ids)
	}

	return result, nil
}
