package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

const (
	CommentMessageAlias = "cm"
)

type CommentMessageListOptions struct {
	Attention    schema.CommentMessageModeratorAttention
	CommentType  schema.CommentMessageType
	PictureItems *PictureItemListOptions
}

func (s *CommentMessageListOptions) Select(db *goqu.Database) (*goqu.SelectDataset, error) {
	sqSelect := db.From(schema.CommentMessageTable.As(CommentMessageAlias))

	return s.Apply(CommentMessageAlias, sqSelect)
}

func (s *CommentMessageListOptions) CountSelect(db *goqu.Database) (*goqu.SelectDataset, error) {
	sqSelect, err := s.Select(db)
	if err != nil {
		return nil, err
	}

	return sqSelect.Select(goqu.COUNT(goqu.Star())), nil
}

func (s *CommentMessageListOptions) Apply(alias string, sqSelect *goqu.SelectDataset) (*goqu.SelectDataset, error) {
	var (
		err        error
		aliasTable = goqu.T(alias)
	)

	sqSelect = sqSelect.Where(
		aliasTable.Col(schema.CommentMessageTableModeratorAttentionColName).Eq(s.Attention),
		aliasTable.Col(schema.CommentMessageTableTypeIDColName).Eq(s.CommentType),
	)

	if s.PictureItems != nil {
		piAlias := AppendPictureItemAlias(alias)

		sqSelect = sqSelect.Join(
			schema.PictureItemTable.As(piAlias),
			goqu.On(
				aliasTable.Col(schema.CommentMessageTableItemIDColName).Eq(
					goqu.T(piAlias).Col(schema.PictureItemTablePictureIDColName),
				),
			),
		)

		sqSelect, err = s.PictureItems.Apply(piAlias, sqSelect)
		if err != nil {
			return nil, err
		}
	}

	return sqSelect, nil
}
