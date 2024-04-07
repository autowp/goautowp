package textstorage

import (
	"context"
	"fmt"

	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

const (
	colID   = "id"
	colText = "text"
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
	sqlSelect := s.db.From(schema.TableTextstorageText).
		Select(colText).
		Where(goqu.C(colID).Eq(id))

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

func (s *Repository) FirstText(ctx context.Context, ids []int64) (string, error) {
	if len(ids) == 0 {
		return "", nil
	}

	args := append([]interface{}{colID}, ids)

	sqlSelect := s.db.From(schema.TableTextstorageText).
		Select(colText).
		Where(
			goqu.C(colID).In(ids),
			goqu.Func("length", goqu.C(colText)).Gt(0),
		).
		Order(goqu.Func("field", args...).Asc()).
		Limit(1)

	result := ""

	success, err := sqlSelect.Executor().ScanValContext(ctx, &result)
	if err != nil {
		return "", err
	}

	if !success {
		return "", fmt.Errorf("text `%v` not found", ids)
	}

	return result, nil
}
