package items

import (
	"context"
	"database/sql"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/pictures"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql" // enable mysql dialect
	"github.com/doug-martin/goqu/v9/exp"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
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
		r, _, err := repository.List(ctx, options)
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
	r, _, err := repository.List(ctx, options)
	require.NoError(t, err)
	require.NotEmpty(t, r)

	for _, i := range r {
		require.NotEmpty(t, i.NameOnly)
	}
}

func createRandomUser(ctx context.Context, t *testing.T, db *goqu.Database) int64 {
	t.Helper()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	emailAddr := "test" + strconv.Itoa(random.Int()) + "@example.com"
	name := "ivan"
	r, err := db.Insert("users").
		Rows(goqu.Record{
			"login":            nil,
			"e_mail":           emailAddr,
			"password":         nil,
			"email_to_check":   nil,
			"hide_e_mail":      1,
			"email_check_code": nil,
			"name":             name,
			"reg_date":         goqu.L("NOW()"),
			"last_online":      goqu.L("NOW()"),
			"timezone":         "Europe/Moscow",
			"last_ip":          goqu.L("INET6_ATON('127.0.0.1')"),
			"language":         "en",
			"role":             "user",
			"uuid":             goqu.L("UUID_TO_BIN(?)", uuid.New().String()),
		}).
		Executor().ExecContext(ctx)
	require.NoError(t, err)

	id, err := r.LastInsertId()
	require.NoError(t, err)

	return id
}

func TestGetUserPicturesBrands(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig("../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)
	ctx := context.Background()

	userID := createRandomUser(ctx, t, goquDB)

	res, err := goquDB.ExecContext(ctx,
		"INSERT INTO item (item_type_id, name, body, produced_exactly) VALUES (?, '', '', 0)",
		BRAND,
	)
	require.NoError(t, err)

	brandID, err := res.LastInsertId()
	require.NoError(t, err)

	res, err = goquDB.ExecContext(ctx,
		"INSERT INTO item (item_type_id, name, body, produced_exactly) VALUES (?, '', '', 0)",
		VEHICLE,
	)
	require.NoError(t, err)

	vehicleID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.ExecContext(ctx,
		"INSERT INTO item_parent (item_id, parent_id, catname) VALUES (?, ?, '')",
		vehicleID, brandID,
	)
	require.NoError(t, err)

	_, err = goquDB.ExecContext(ctx,
		"INSERT INTO item_parent_cache (item_id, parent_id, diff) VALUES (?, ?, 0), (?, ?, 0), (?, ?, 1)",
		brandID, brandID, vehicleID, vehicleID, vehicleID, brandID,
	)
	require.NoError(t, err)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	identity := "t" + strconv.Itoa(int(random.Uint32()%100000))

	res, err = goquDB.ExecContext(ctx,
		"INSERT INTO pictures (identity, status, ip, owner_id) VALUES (?, 'accepted', '', ?)",
		identity, userID,
	)
	require.NoError(t, err)

	pictureID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.ExecContext(ctx,
		"INSERT INTO picture_item (picture_id, item_id) VALUES (?, ?)",
		pictureID, vehicleID,
	)
	require.NoError(t, err)

	repository := NewRepository(goquDB)
	options := ListOptions{
		Language: "en",
		Fields: ListFields{
			NameOnly:             true,
			CurrentPicturesCount: true,
		},
		DescendantPictures: &ItemPicturesOptions{
			Pictures: &PicturesOptions{
				OwnerID: userID,
				Status:  pictures.StatusAccepted,
			},
		},
		TypeID:     []ItemType{BRAND},
		Limit:      10,
		SortByName: true,
	}
	r, _, err := repository.List(ctx, options)
	require.NoError(t, err)
	require.NotEmpty(t, r)

	for _, i := range r {
		require.Equal(t, int32(1), i.CurrentPicturesCount)
	}
}

func TestPaginator(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig("../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)
	ctx := context.Background()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	name := "t" + strconv.Itoa(int(random.Uint32()%100000))

	for i := 0; i < 10; i++ {
		res, err := goquDB.ExecContext(ctx,
			"INSERT INTO item (item_type_id, name, body, produced_exactly) VALUES (?, ?, '', 0)",
			BRAND, name+"_"+strconv.Itoa(i),
		)
		require.NoError(t, err)

		itemID, err := res.LastInsertId()
		require.NoError(t, err)

		_, err = goquDB.ExecContext(ctx,
			"INSERT INTO item_language (item_id, language, name) VALUES (?, ?, ?)",
			itemID, "en", name+"_"+strconv.Itoa(i),
		)
		require.NoError(t, err)
	}

	repository := NewRepository(goquDB)
	options := ListOptions{
		Language: "en",
		Limit:    2,
		Page:     2,
		Name:     name + "%",
	}
	r, pages, err := repository.List(ctx, options)
	require.NoError(t, err)
	require.NotEmpty(t, r)
	require.Equal(t, 2, len(r))
	require.Equal(t, int32(10), pages.TotalItemCount)
	require.Equal(t, int32(5), pages.PageCount)
	require.Equal(t, int32(2), pages.Current)
}
