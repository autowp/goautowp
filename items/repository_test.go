package items

import (
	"database/sql"
	"github.com/autowp/goautowp/config"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTopBrandsListRu(t *testing.T) {
	cfg := config.LoadConfig("../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	repository := NewRepository(db)
	r, err := repository.TopBrandList("ru")
	require.NoError(t, err)
	require.NotEmpty(t, r)
	require.NotEmpty(t, r.Brands)
	require.Greater(t, r.Total, 0)
}

func TestTopBrandsListZh(t *testing.T) {
	cfg := config.LoadConfig("../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	repository := NewRepository(db)
	r, err := repository.TopBrandList("zh")
	require.NoError(t, err)
	require.NotEmpty(t, r)
	require.NotEmpty(t, r.Brands)
	require.Greater(t, r.Total, 0)
}
