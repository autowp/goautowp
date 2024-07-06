package goautowp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"image"
	_ "image/jpeg" // support JPEG decoding.
	_ "image/png"  // support PNG decoding.
	"io"
	"net/http"
	"strconv"

	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/corona10/goimagehash"
	"github.com/doug-martin/goqu/v9"
	"github.com/sirupsen/logrus"
)

var errInvalidID = errors.New("invalid id provided")

const (
	threshold = 3
	decimal   = 10
)

// DuplicateFinder Main Object.
type DuplicateFinder struct {
	db *goqu.Database
}

// DuplicateFinderInputMessage InputMessage.
type DuplicateFinderInputMessage struct {
	PictureID int    `json:"picture_id"`
	URL       string `json:"url"`
}

// NewDuplicateFinder constructor.
func NewDuplicateFinder(db *goqu.Database) (*DuplicateFinder, error) {
	s := &DuplicateFinder{
		db: db,
	}

	return s, nil
}

// ListenAMQP for incoming messages.
func (s *DuplicateFinder) ListenAMQP(ctx context.Context, url string, queue string, quitChan chan bool) error {
	rabbitMQ, err := util.ConnectRabbitMQ(url)
	if err != nil {
		logrus.Error(err)

		return err
	}

	ch, err := rabbitMQ.Channel()
	if err != nil {
		return err
	}
	defer util.Close(ch)

	inQ, err := ch.QueueDeclare(
		queue, // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		inQ.Name, // queue
		"",       // consumer
		true,     // auto-ack
		false,    // exclusive
		false,    // no-local
		false,    // no-wait
		nil,      // args
	)
	if err != nil {
		return err
	}

	done := false
	for !done {
		select {
		case <-quitChan:
			logrus.Info("DuplicateFinder got quit signal")

			done = true

			break
		case msg := <-msgs:
			if msg.ContentType != "application/json" {
				logrus.Errorf("unexpected mime `%v`", msg.ContentType)

				continue
			}

			var message DuplicateFinderInputMessage

			err := json.Unmarshal(msg.Body, &message)
			if err != nil {
				logrus.Errorf("failed to parse json `%s`: %s", err.Error(), msg.Body)

				continue
			}

			err = s.Index(ctx, message.PictureID, message.URL)
			if err != nil {
				logrus.Error(err)
			}
		}
	}

	logrus.Info("Disconnecting RabbitMQ")

	return rabbitMQ.Close()
}

// Index picture image
// #nosec G107
func (s *DuplicateFinder) Index(ctx context.Context, id int, url string) error {
	logrus.Infof("Indexing picture %v", id)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose
	if err != nil {
		return err
	}
	defer util.Close(resp.Body)

	logrus.Infof("Calculate hash for %v", url)

	hash, err := getFileHash(resp.Body)
	if err != nil {
		return err
	}

	_, err = s.db.Insert(schema.DfHashTable).Rows(goqu.Record{
		schema.DfHashTablePictureIDColName: id,
		// can't use uint64 directly because of mysql driver issue
		schema.DfHashTableHashColName: goqu.L(strconv.FormatUint(hash, 10)),
	}).Executor().Exec()
	if err != nil {
		return err
	}

	return s.updateDistance(ctx, id)
}

func getFileHash(reader io.Reader) (uint64, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return 0, err
	}

	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return 0, err
	}

	return hash.GetHash(), nil
}

func (s *DuplicateFinder) updateDistance(ctx context.Context, id int) error {
	if id <= 0 {
		return errInvalidID
	}

	var hash uint64

	success, err := s.db.Select(schema.DfHashTableHashCol).
		From(schema.DfHashTable).
		Where(schema.DfHashTablePictureIDCol.Eq(id)).
		ScanValContext(ctx, &hash)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	const alias = "distance"

	var sts []struct {
		PictureID int `db:"picture_id"`
		Distance  int `db:"distance"`
	}

	err = s.db.Select(
		schema.DfHashTablePictureIDCol,
		goqu.Func("BIT_COUNT", goqu.L("? ^ "+strconv.FormatUint(hash, decimal), schema.DfHashTableHashCol)).As(alias),
	).
		From(schema.DfHashTable).
		Where(schema.DfHashTablePictureIDCol.Neq(id)).
		Having(goqu.C(alias).Lte(threshold)).
		Executor().ScanStructsContext(ctx, &sts)

	if len(sts) == 0 {
		return nil
	}

	if err != nil {
		return err
	}

	records := make([]goqu.Record, 0, len(sts)*2)

	for _, st := range sts {
		records = append(records, goqu.Record{
			schema.DfDistanceTableSrcPictureIDColName: id,
			schema.DfDistanceTableDstPictureIDColName: st.PictureID,
			schema.DfDistanceTableDistanceColName:     st.Distance,
		}, goqu.Record{
			schema.DfDistanceTableSrcPictureIDColName: st.PictureID,
			schema.DfDistanceTableDstPictureIDColName: id,
			schema.DfDistanceTableDistanceColName:     st.Distance,
		})
	}

	_, err = s.db.Insert(schema.DfDistanceTable).
		Rows(records).
		OnConflict(goqu.DoUpdate(
			schema.DfDistanceTableSrcPictureIDColName+","+schema.DfDistanceTableDstPictureIDColName, goqu.Record{
				schema.DfDistanceTableDistanceColName: goqu.Func("VALUES", goqu.C(schema.DfDistanceTableDistanceColName)),
			})).
		Executor().ExecContext(ctx)

	return err
}
