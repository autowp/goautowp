package attrs

import (
	"context"
	"database/sql"
	"testing"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/textstorage"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql" // enable mysql dialect
	_ "github.com/go-sql-driver/mysql"               // enable mysql driver
	"github.com/stretchr/testify/require"
)

func createRepository(t *testing.T) *Repository {
	t.Helper()

	cfg := config.LoadConfig("..")

	autowpDB, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", autowpDB)

	i18n, err := i18nbundle.New()
	require.NoError(t, err)

	textstorageRepository := textstorage.New(goquDB)
	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	itemsRepository := items.NewRepository(goquDB, 0, cfg.ContentLanguages, textstorageRepository)
	picturesRepository := pictures.NewRepository(goquDB, imageStorage, textstorageRepository, itemsRepository)

	repo := NewRepository(goquDB, i18n, itemsRepository, picturesRepository, imageStorage)

	return repo
}

func TestAttributes(t *testing.T) {
	t.Parallel()

	repo := createRepository(t)

	ctx := context.Background()

	_, err := repo.Attributes(ctx, 0, 0)
	require.NoError(t, err)

	rows, err := repo.Attributes(ctx, 0, 95)
	require.NoError(t, err)
	require.NotEmpty(t, rows)
}

func TestAttributeTypes(t *testing.T) {
	t.Parallel()

	repo := createRepository(t)

	ctx := context.Background()

	_, err := repo.AttributeTypes(ctx)
	require.NoError(t, err)
}

func TestUnits(t *testing.T) {
	t.Parallel()

	repo := createRepository(t)

	ctx := context.Background()

	_, err := repo.Units(ctx)
	require.NoError(t, err)
}

func TestZones(t *testing.T) {
	t.Parallel()

	repo := createRepository(t)

	ctx := context.Background()

	_, err := repo.Zones(ctx)
	require.NoError(t, err)
}
