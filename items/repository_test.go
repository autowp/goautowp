package items

import (
	"database/sql"
	"github.com/autowp/goautowp/config"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTopBrandsListRu(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig("../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	repository := NewRepository(goquDB)
	options := ItemsOptions{
		Language: "ru",
		Fields: ListFields{
			Name:                true,
			DescendantsCount:    true,
			NewDescendantsCount: true,
		},
		TypeID:     []ItemType{BRAND},
		Limit:      TopBrandsCount,
		OrderBy:    "descendants_count DESC",
		SortByName: true,
	}
	r, err := repository.List(options)
	require.NoError(t, err)
	require.NotEmpty(t, r)
	require.NotEmpty(t, r)

	c, err := repository.Count(options)
	require.NoError(t, err)
	require.Greater(t, c, 0)
}

func TestTopBrandsListZh(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig("../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	repository := NewRepository(goquDB)
	options := ItemsOptions{
		Language: "zh",
		Fields: ListFields{
			Name:                true,
			DescendantsCount:    true,
			NewDescendantsCount: true,
		},
		TypeID:     []ItemType{BRAND},
		Limit:      TopBrandsCount,
		OrderBy:    "descendants_count DESC",
		SortByName: true,
	}
	r, err := repository.List(options)
	require.NoError(t, err)
	require.NotEmpty(t, r)
	require.NotEmpty(t, r)

	c, err := repository.Count(options)
	require.NoError(t, err)
	require.Greater(t, c, 0)
}
