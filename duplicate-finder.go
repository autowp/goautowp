package goautowp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // support JPEG decoding
	_ "image/png"  // support PNG decoding
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/autowp/goautowp/util"
	"github.com/corona10/goimagehash"
	"github.com/getsentry/sentry-go"
	"github.com/streadway/amqp"
)

const threshold = 3

// DuplicateFinder Main Object
type DuplicateFinder struct {
	db *sql.DB
}

// DuplicateFinderInputMessage InputMessage
type DuplicateFinderInputMessage struct {
	PictureID int    `json:"picture_id"`
	URL       string `json:"url"`
}

// NewDuplicateFinder constructor
func NewDuplicateFinder(db *sql.DB) (*DuplicateFinder, error) {

	s := &DuplicateFinder{
		db: db,
	}

	return s, nil
}

func connectRabbitMQ(config string) (*amqp.Connection, error) {
	start := time.Now()
	timeout := 60 * time.Second

	log.Println("Waiting for rabbitMQ")

	var rabbitMQ *amqp.Connection
	var err error
	for {
		rabbitMQ, err = amqp.Dial(config)
		if err == nil {
			log.Println("Started.")
			break
		}

		if time.Since(start) > timeout {
			return nil, err
		}

		fmt.Print(".")
		time.Sleep(100 * time.Millisecond)
	}

	return rabbitMQ, nil
}

// Listen for incoming messages
func (s *DuplicateFinder) ListenAMQP(url string, queue string, quitChan chan bool) error {

	rabbitMQ, err := connectRabbitMQ(url)
	if err != nil {
		log.Println(err)
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
			log.Println("DuplicateFinder got quit signal")
			done = true
			break
		case d := <-msgs:
			if d.ContentType != "application/json" {
				sentry.CaptureException(fmt.Errorf("unexpected mime `%v`", d.ContentType))
				continue
			}

			var message DuplicateFinderInputMessage
			err := json.Unmarshal(d.Body, &message)
			if err != nil {
				sentry.CaptureException(fmt.Errorf("failed to parse json `%v`: %s", err, d.Body))
				continue
			}

			err = s.Index(message.PictureID, message.URL)
			if err != nil {
				sentry.CaptureException(err)
			}
		}
	}

	log.Println("Disconnecting RabbitMQ")
	return rabbitMQ.Close()
}

// Index picture image
// #nosec G107
func (s *DuplicateFinder) Index(id int, url string) error {
	log.Printf("Indexing picture %v\n", id)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer util.Close(resp.Body)

	log.Printf("Calculate hash for %v\n", url)

	hash, err := getFileHash(resp.Body)
	if err != nil {
		return err
	}

	stmt, err := s.db.Prepare(`
		INSERT INTO df_hash (picture_id, hash)
		VALUES (?, ?)
	`)
	if err != nil {
		return err
	}
	defer util.Close(stmt)

	_, err = stmt.Exec(id, hash)
	if err != nil {
		return err
	}

	return s.updateDistance(id)
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

func (s *DuplicateFinder) updateDistance(id int) error {
	if id <= 0 {
		return errors.New("invalid id provided")
	}

	var hash uint64
	err := s.db.QueryRow("SELECT hash FROM df_hash WHERE picture_id = ?", id).Scan(&hash)
	if err != nil {
		return err
	}

	insertStmt, err := s.db.Prepare(`
		INSERT INTO df_distance (src_picture_id, dst_picture_id, distance)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE distance=distance;
	`)
	if err != nil {
		return err
	}
	defer util.Close(insertStmt)

	// nolint: gosec
	rows, err := s.db.Query(`
		SELECT picture_id, BIT_COUNT(hash ^ `+strconv.FormatUint(hash, 10)+`) AS distance
		FROM df_hash 
		WHERE picture_id != ? 
		HAVING distance <= ?
	`, id, threshold)
	if err != nil {
		return err
	}
	if err == sql.ErrNoRows {
		return nil
	}

	defer util.Close(rows)

	for rows.Next() {
		var pictureID int
		var distance int
		serr := rows.Scan(&pictureID, &distance)
		if serr != nil {
			return serr
		}

		_, serr = insertStmt.Exec(id, pictureID, distance)
		if serr != nil {
			return serr
		}

		_, serr = insertStmt.Exec(pictureID, id, distance)
		if serr != nil {
			return serr
		}
	}

	return nil
}
