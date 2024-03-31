package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/itemofday"
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
)

func TestYoomoneyWebhookInvalidLabel(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	itemOfDayRepository := itemofday.NewRepository(goquDB)

	ctx := context.Background()

	yh, err := NewYoomoneyHandler("0.99", "01234567890ABCDEF01234567890", itemOfDayRepository)
	require.NoError(t, err)

	err = yh.Handle(ctx, YoomoneyWebhook{
		NotificationType: "p2p-incoming",
		OperationID:      "1234567",
		Amount:           "300.00",
		WithdrawAmount:   "1.00",
		Currency:         "643",
		Datetime:         "2011-07-01T09:00:00.000+04:00",
		Sender:           "41001XXXXXXXX",
		Codepro:          "false",
		Label:            "YM.label.12345",
		SHA1Hash:         "a2ee4a9195f4a90e893cff4f62eeba0b662321f9",
		TestNotification: false,
		Unaccepted:       false,
	})

	require.ErrorContains(t, err, "label not matched by regular expression")
}

func TestYoomoneyWebhookHappyPath(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	itemOfDayRepository := itemofday.NewRepository(goquDB)
	itemOfDayRepository.SetMinPictures(0)

	ctx := context.Background()

	yh, err := NewYoomoneyHandler("0.99", "01234567890ABCDEF01234567890", itemOfDayRepository)
	require.NoError(t, err)

	dateStr := time.Now().Format(itemofday.YoomoneyLabelDateFormat)

	// prepare test data
	_, err = goquDB.Delete("of_day").Where(goqu.C("day_date").Eq(dateStr)).Executor().Exec()
	require.NoError(t, err)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	//nolint:gosec
	r1, err := db.ExecContext(
		ctx, `
			INSERT INTO `+schema.TableItem+` (name, is_group, item_type_id, catname, body, produced_exactly)
			VALUES (?, 0, 5, ?, '', 0)
		`, fmt.Sprintf("item-of-day-%d", random.Int()), fmt.Sprintf("brand1-%d", random.Int()),
	)
	require.NoError(t, err)

	itemID, err := r1.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.TableItemParentCache).Rows(goqu.Record{
		"item_id":   itemID,
		"parent_id": itemID,
		"diff":      0,
	}).Executor().Exec()
	require.NoError(t, err)

	identity := "t" + strconv.Itoa(int(random.Uint32()%100000))

	res, err := goquDB.ExecContext(ctx,
		"INSERT INTO "+schema.TablePicture+" (identity, status, ip, owner_id) VALUES (?, 'accepted', '', null)",
		identity,
	)
	require.NoError(t, err)

	pictureID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.ExecContext(ctx,
		"INSERT INTO "+schema.TablePictureItem+" (picture_id, item_id) VALUES (?, ?)",
		pictureID, itemID,
	)
	require.NoError(t, err)

	// check prepared item passes candidate checks
	r := itemofday.CandidateRecord{}

	sqSelect := itemOfDayRepository.CandidateQuery()
	_, err = sqSelect.Limit(1).Executor().ScanStructContext(ctx, &r)
	require.NoError(t, err)
	require.NotZero(t, itemID)

	label := fmt.Sprintf("vod/%s/%d/0", dateStr, itemID)

	fields := YoomoneyWebhook{
		NotificationType: "p2p-incoming",
		OperationID:      "1234567",
		Amount:           "300.00",
		WithdrawAmount:   "1.00",
		Currency:         "643",
		Datetime:         "2011-07-01T09:00:00.000+04:00",
		Sender:           "41001XXXXXXXX",
		Codepro:          "false",
		Label:            label,
		TestNotification: false,
		Unaccepted:       false,
	}
	fields.SHA1Hash, err = yh.Hash(fields)
	require.NoError(t, err)

	err = yh.Handle(ctx, fields)

	require.NoError(t, err)
}
