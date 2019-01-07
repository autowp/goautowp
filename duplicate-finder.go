package goautowp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // support JPEG decoding
	_ "image/png"  // support PNG decoding
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/autowp/goautowp/util"
	"github.com/corona10/goimagehash"
	"github.com/streadway/amqp"
)

const threshold = 3

// DuplicateFinder Main Object
type DuplicateFinder struct {
	db    *sql.DB
	queue string
	conn  *amqp.Connection
	quit  chan bool
	// logger    *util.Logger
	imagesDir string
	logger    *util.Logger
}

// DuplicateFinderInputMessage InputMessage
type DuplicateFinderInputMessage struct {
	PictureID int `json:"picture_id"`
}

// NewDuplicateFinder constructor
func NewDuplicateFinder(
	wg *sync.WaitGroup,
	db *sql.DB,
	rabbitmMQ *amqp.Connection,
	queue string,
	imagesDir string,
	logger *util.Logger,
) (*DuplicateFinder, error) {
	s := &DuplicateFinder{
		db:        db,
		conn:      rabbitmMQ,
		queue:     queue,
		quit:      make(chan bool),
		logger:    logger,
		imagesDir: imagesDir,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("DuplicateFinder listener started")
		err := s.listen()
		if err != nil {
			s.logger.Fatal(err)
		}
		log.Println("DuplicateFinder listener stopped")
	}()

	return s, nil
}

// Close all connections
func (s *DuplicateFinder) Close() {

	s.quit <- true
	close(s.quit)
}

// Listen for incoming messages
func (s *DuplicateFinder) listen() error {
	if s.conn == nil {
		return fmt.Errorf("RabbitMQ connection not initialized")
	}

	ch, err := s.conn.Channel()
	if err != nil {
		return err
	}
	defer util.Close(ch)

	inQ, err := ch.QueueDeclare(
		s.queue, // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
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

	quit := false
	for !quit {
		select {
		case <-s.quit:
			quit = true
			return nil
		case d := <-msgs:
			if d.ContentType != "application/json" {
				s.logger.Warning(fmt.Errorf("unexpected mime `%v`", d.ContentType))
				return nil
				// continue
			}

			var message DuplicateFinderInputMessage
			err := json.Unmarshal(d.Body, &message)
			if err != nil {
				s.logger.Warning(fmt.Errorf("failed to parse json `%v`: %s", err, d.Body))
				continue
			}

			err = s.Index(message.PictureID)
			if err != nil {
				s.logger.Warning(err)
			}
		}
	}

	return nil
}

// Index picture image
func (s *DuplicateFinder) Index(id int) error {
	log.Printf("Indexing picture %v\n", id)

	var imageID int
	err := s.db.QueryRow("SELECT image_id FROM pictures WHERE id = ?", id).Scan(&imageID)
	if err != nil {
		return err
	}

	var filepath string
	err = s.db.QueryRow("SELECT filepath FROM image WHERE id = ?", imageID).Scan(&filepath)
	if err != nil {
		return err
	}

	log.Printf("Calculate hash for %v\n", filepath)

	hash, err := getFileHash(s.imagesDir + "/" + filepath)
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

func getFileHash(fp string) (uint64, error) {
	if fp == "" {
		return 0, errors.New("Invalid filepath")
	}

	file, err := os.Open(filepath.Clean(fp))
	if err != nil {
		return 0, err
	}
	defer util.Close(file)
	img, _, err := image.Decode(file)
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
		return errors.New("Invalid id provided")
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
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	defer util.Close(rows)

	for rows.Next() {
		var pictureID int
		var distance int
		if serr := rows.Scan(&pictureID, &distance); serr != nil {
			return serr
		}

		_, serr := insertStmt.Exec(id, pictureID, distance)
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

// ImagesDir ImagesDir
func (s *DuplicateFinder) ImagesDir() string {
	return s.imagesDir
}
