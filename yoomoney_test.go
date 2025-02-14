package goautowp

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/autowp/goautowp/itemofday"
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
)

func TestYoomoneyWebhookInvalidLabel(t *testing.T) {
	t.Parallel()

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	itemOfDayRepository := itemofday.NewRepository(goquDB)

	ctx := t.Context()

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

	goquDB, err := cnt.GoquDB()
	require.NoError(t, err)

	itemOfDayRepository := itemofday.NewRepository(goquDB)
	itemOfDayRepository.SetMinPictures(0)

	ctx := t.Context()

	yh, err := NewYoomoneyHandler("0.99", "01234567890ABCDEF01234567890", itemOfDayRepository)
	require.NoError(t, err)

	dateStr := time.Now().Format(itemofday.YoomoneyLabelDateFormat)

	// prepare test data
	_, err = goquDB.Delete(schema.OfDayTable).Where(schema.OfDayTableDayDateCol.Eq(dateStr)).Executor().Exec()
	require.NoError(t, err)

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	r1, err := goquDB.Insert(schema.ItemTable).Rows(goqu.Record{
		schema.ItemTableNameColName:            fmt.Sprintf("item-of-day-%d", random.Int()),
		schema.ItemTableIsGroupColName:         0,
		schema.ItemTableItemTypeIDColName:      schema.ItemTableItemTypeIDBrand,
		schema.ItemTableCatnameColName:         fmt.Sprintf("brand1-%d", random.Int()),
		schema.ItemTableBodyColName:            "",
		schema.ItemTableProducedExactlyColName: 0,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	itemID, err := r1.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.ItemParentCacheTable).Rows(goqu.Record{
		schema.ItemParentCacheTableItemIDColName:   itemID,
		schema.ItemParentCacheTableParentIDColName: itemID,
		schema.ItemParentCacheTableDiffColName:     0,
	}).Executor().Exec()
	require.NoError(t, err)

	identity := "t" + strconv.Itoa(int(random.Uint32()%100000))

	res, err := goquDB.Insert(schema.PictureTable).Rows(goqu.Record{
		schema.PictureTableIdentityColName: identity,
		schema.PictureTableStatusColName:   schema.PictureStatusAccepted,
		schema.PictureTableIPColName:       "",
		schema.PictureTableOwnerIDColName:  nil,
	}).Executor().ExecContext(ctx)
	require.NoError(t, err)

	pictureID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = goquDB.Insert(schema.PictureItemTable).Rows(goqu.Record{
		schema.PictureItemTablePictureIDColName: pictureID,
		schema.PictureItemTableItemIDColName:    itemID,
	}).Executor().ExecContext(ctx)
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
