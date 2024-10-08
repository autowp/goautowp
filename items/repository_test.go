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
	"github.com/autowp/goautowp/schema"
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

	repository := NewRepository(goquDB, 200)

	langs := []string{"ru", "zh"}

	for _, lang := range langs {
		options := ListOptions{
			Language: lang,
			Fields: ListFields{
				NameOnly:                   true,
				DescendantsCount:           true,
				NewDescendantsCount:        true,
				Description:                true,
				FullText:                   true,
				NameHTML:                   true,
				NameText:                   true,
				NameDefault:                true,
				ItemsCount:                 true,
				NewItemsCount:              true,
				DescendantTwinsGroupsCount: true,
			},
			TypeID:     []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
			Limit:      150,
			OrderBy:    []exp.OrderedExpression{goqu.C("descendants_count").Desc()},
			SortByName: true,
		}
		r, _, err := repository.List(ctx, options, true)
		require.NoError(t, err)
		require.NotEmpty(t, r)

		c, err := repository.Count(ctx, options)
		require.NoError(t, err)
		require.Positive(t, c)
	}
}

func TestListFilters(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig("../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)
	ctx := context.Background()

	repository := NewRepository(goquDB, 200)

	options := ListOptions{
		Language: "en",
		Fields: ListFields{
			NameOnly:                   true,
			DescendantsCount:           true,
			NewDescendantsCount:        true,
			Description:                true,
			FullText:                   true,
			NameHTML:                   true,
			NameText:                   true,
			NameDefault:                true,
			ItemsCount:                 true,
			NewItemsCount:              true,
			DescendantTwinsGroupsCount: true,
		},
		TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
		ChildItems: &ParentItemsListOptions{
			ChildItems: &ListOptions{
				TypeID:       []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDVehicle},
				IsConcept:    true,
				EngineItemID: 1,
			},
		},
		ParentItems: &ParentItemsListOptions{
			ParentItems: &ListOptions{
				TypeID:    []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDVehicle},
				NoParents: true,
				Catname:   "test",
			},
		},
		Limit:   150,
		OrderBy: []exp.OrderedExpression{goqu.C("descendants_count").Desc()},
	}
	_, _, err = repository.List(ctx, options, true)
	require.NoError(t, err)
}

func TestGetItemsNameAndCatnameShouldNotBeOmittedWhenDescendantsCountRequested(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig("../")
	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)
	ctx := context.Background()

	repository := NewRepository(goquDB, 200)
	options := ListOptions{
		Language: "en",
		Fields: ListFields{
			NameOnly:         true,
			DescendantsCount: true,
			ChildsCount:      true,
		},
		TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
		Limit:  10,
	}
	_, _, err = repository.List(ctx, options, true)
	require.NoError(t, err)
}

func createRandomUser(ctx context.Context, t *testing.T, db *goqu.Database) int64 {
	t.Helper()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	emailAddr := "test" + strconv.Itoa(random.Int()) + "@example.com"
	name := "ivan"
	res, err := db.Insert(schema.UserTable).
		Rows(goqu.Record{
			schema.UserTableLoginColName:          nil,
			schema.UserTableEmailColName:          emailAddr,
			schema.UserTablePasswordColName:       nil,
			schema.UserTableEmailToCheckColName:   nil,
			schema.UserTableHideEmailColName:      1,
			schema.UserTableEmailCheckCodeColName: nil,
			schema.UserTableNameColName:           name,
			schema.UserTableRegDateColName:        goqu.Func("NOW"),
			schema.UserTableLastOnlineColName:     goqu.Func("NOW"),
			schema.UserTableTimezoneColName:       "Europe/Moscow",
			schema.UserTableLastIPColName:         goqu.Func("INET6_ATON", "127.0.0.1"),
			schema.UserTableLanguageColName:       "en",
			schema.UserTableRoleColName:           "user",
			schema.UserTableUUIDColName:           goqu.Func("UUID_TO_BIN", uuid.New().String()),
		}).
		Executor().ExecContext(ctx)
	require.NoError(t, err)

	id, err := res.LastInsertId()
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

	res, err := goquDB.Insert(schema.ItemTable).Rows(goqu.Record{
		schema.ItemTableItemTypeIDColName:      schema.ItemTableItemTypeIDBrand,
		schema.ItemTableNameColName:            "",
		schema.ItemTableBodyColName:            "",
		schema.ItemTableProducedExactlyColName: 0,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	brandID, err := res.LastInsertId()
	require.NoError(t, err)

	res, err = goquDB.Insert(schema.ItemTable).Rows(goqu.Record{
		schema.ItemTableItemTypeIDColName:      schema.ItemTableItemTypeIDVehicle,
		schema.ItemTableNameColName:            "",
		schema.ItemTableBodyColName:            "",
		schema.ItemTableProducedExactlyColName: 0,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	vehicleID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.ItemParentTable).Rows(goqu.Record{
		"item_id":   vehicleID,
		"parent_id": brandID,
		"catname":   "",
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.ItemParentCacheTable).
		Cols(
			schema.ItemParentCacheTableItemIDColName,
			schema.ItemParentCacheTableParentIDColName,
			schema.ItemParentCacheTableDiffColName,
		).
		Vals(
			goqu.Vals{brandID, brandID, 0},
			goqu.Vals{vehicleID, vehicleID, 0},
			goqu.Vals{vehicleID, brandID, 1},
		).
		Executor().ExecContext(ctx)
	require.NoError(t, err)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	identity := "t" + strconv.Itoa(int(random.Uint32()%100000))

	res, err = goquDB.Insert(schema.PictureTable).Rows(goqu.Record{
		schema.PictureTableIdentityColName: identity,
		schema.PictureTableStatusColName:   "accepted",
		schema.PictureTableIPColName:       "",
		schema.PictureTableOwnerIDColName:  userID,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	pictureID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.PictureItemTable).Rows(goqu.Record{
		schema.PictureItemTablePictureIDColName: pictureID,
		schema.PictureItemTableItemIDColName:    vehicleID,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	repository := NewRepository(goquDB, 200)
	options := ListOptions{
		Language: "en",
		Fields: ListFields{
			NameOnly:             true,
			CurrentPicturesCount: true,
			ChildsCount:          true,
		},
		DescendantPictures: &ItemPicturesOptions{
			Pictures: &PicturesOptions{
				OwnerID: userID,
				Status:  pictures.StatusAccepted,
			},
		},
		TypeID:     []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
		Limit:      10,
		SortByName: true,
	}
	r, _, err := repository.List(ctx, options, true)
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

	for i := range 10 {
		res, err := goquDB.Insert(schema.ItemTable).Rows(goqu.Record{
			schema.ItemTableItemTypeIDColName:      schema.ItemTableItemTypeIDBrand,
			schema.ItemTableNameColName:            name + "_" + strconv.Itoa(i),
			schema.ItemTableBodyColName:            "",
			schema.ItemTableProducedExactlyColName: 0,
		}).Executor().ExecContext(ctx)
		require.NoError(t, err)

		itemID, err := res.LastInsertId()
		require.NoError(t, err)

		_, err = goquDB.Insert(schema.ItemLanguageTable).Rows(goqu.Record{
			schema.ItemLanguageTableItemIDColName:   itemID,
			schema.ItemLanguageTableLanguageColName: "en",
			schema.ItemLanguageTableNameColName:     name + "_" + strconv.Itoa(i),
		}).Executor().ExecContext(ctx)
		require.NoError(t, err)
	}

	repository := NewRepository(goquDB, 200)
	options := ListOptions{
		Language: "en",
		Limit:    2,
		Page:     2,
		Name:     name + "%",
	}
	r, pages, err := repository.List(ctx, options, true)
	require.NoError(t, err)
	require.NotEmpty(t, r)
	require.Len(t, r, 2)
	require.Equal(t, int32(10), pages.TotalItemCount)
	require.Equal(t, int32(5), pages.PageCount)
	require.Equal(t, int32(2), pages.Current)
}
