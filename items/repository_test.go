package items

import (
	"context"
	"database/sql"
	"testing"

	"github.com/autowp/goautowp/config"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql" // enable mysql dialect
	"github.com/doug-martin/goqu/v9/exp"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestTopBrandsListRuZh(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig("../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)
	ctx := context.Background()

	repository := NewRepository(goquDB)

	langs := []string{"ru", "zh"}

	for _, lang := range langs {
		options := ListOptions{
			Language: lang,
			Fields: ListFields{
				NameOnly:            true,
				DescendantsCount:    true,
				NewDescendantsCount: true,
				Description:         true,
				FullText:            true,
				NameHTML:            true,
				NameText:            true,
				NameDefault:         true,
				ItemsCount:          true,
				NewItemsCount:       true,
			},
			TypeID:     []ItemType{BRAND},
			Limit:      TopBrandsCount,
			OrderBy:    []exp.OrderedExpression{goqu.I("descendants_count").Desc()},
			SortByName: true,
		}
		r, err := repository.List(ctx, options)
		require.NoError(t, err)
		require.NotEmpty(t, r)

		c, err := repository.Count(ctx, options)
		require.NoError(t, err)
		require.Greater(t, c, 0)
	}
}

func TestGetItemsNameAndCatnameShouldNotBeOmittedWhenDescendantsCountRequested(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig("../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)
	ctx := context.Background()

	repository := NewRepository(goquDB)
	options := ListOptions{
		Language: "en",
		Fields: ListFields{
			NameOnly:         true,
			DescendantsCount: true,
		},
		TypeID: []ItemType{BRAND},
		Limit:  10,
	}
	r, err := repository.List(ctx, options)
	require.NoError(t, err)
	require.NotEmpty(t, r)

	for _, i := range r {
		require.NotEmpty(t, i.NameOnly)
		require.NotEmpty(t, i.Catname)
	}
}
