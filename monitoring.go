package goautowp

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
	"net"
	"strings"
	"time"

	"github.com/autowp/goautowp/util"
)

// Monitoring Main Object
type Monitoring struct {
	db *pgxpool.Pool
}

// MonitoringInputMessage InputMessage
type MonitoringInputMessage struct {
	IP        net.IP    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}

// ListOfTopItem ListOfTopItem
type ListOfTopItem struct {
	IP    net.IP `json:"ip"`
	Count int    `json:"count"`
}

// NewMonitoring constructor
func NewMonitoring(db *pgxpool.Pool) (*Monitoring, error) {
	s := &Monitoring{
		db: db,
	}

	return s, nil
}

// Listen for incoming messages
func (s *Monitoring) Listen(url string, queue string, quitChan chan bool) error {

	conn, err := connectRabbitMQ(url)
	if err != nil {
		logrus.Error(err)
		return err
	}

	ch, err := conn.Channel()
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

	quit := false
	for !quit {
		select {
		case d := <-msgs:
			if d.ContentType != "application/json" {
				logrus.Errorf("unexpected mime `%s`", d.ContentType)
				continue
			}

			var message MonitoringInputMessage
			err = json.Unmarshal(d.Body, &message)
			if err != nil {
				logrus.Errorf("failed to parse json `%v`: %s", err, d.Body)
				continue
			}

			err = s.Add(message.IP, message.Timestamp)
			if err != nil {
				logrus.Error(err.Error())
			}

		case <-quitChan:
			quit = true
		}
	}

	logrus.Info("Disconnecting RabbitMQ")
	return conn.Close()

}

// Add item to Monitoring
func (s *Monitoring) Add(ip net.IP, timestamp time.Time) error {

	_, err := s.db.Exec(context.Background(), `
		INSERT INTO ip_monitoring (day_date, hour, tenminute, minute, ip, count)
		VALUES (
			$1::timestamptz,
			EXTRACT(HOUR FROM $1::timestamptz),
			FLOOR(EXTRACT(MINUTE FROM $1::timestamptz)/10),
			EXTRACT(MINUTE FROM $1::timestamptz),
			$2,
			1
		)
		ON CONFLICT(ip,day_date,hour,tenminute,minute) DO UPDATE SET count=ip_monitoring.count+1
	`, timestamp, ip.String())

	return err
}

// GC Garbage Collect
func (s *Monitoring) GC() (int64, error) {

	ct, err := s.db.Exec(context.Background(), "DELETE FROM ip_monitoring WHERE day_date < CURRENT_DATE")
	if err != nil {
		return 0, err
	}

	affected := ct.RowsAffected()

	return affected, nil
}

// Clear removes all collected data
func (s *Monitoring) Clear() error {
	_, err := s.db.Exec(context.Background(), "DELETE FROM ip_monitoring")
	return err
}

// ClearIP removes all data collected for IP
func (s *Monitoring) ClearIP(ip net.IP) error {
	logrus.Info(ip.String() + ": clear monitoring")
	_, err := s.db.Exec(context.Background(), "DELETE FROM ip_monitoring WHERE ip = $1", ip.String())

	return err
}

// ListOfTop ListOfTop
func (s *Monitoring) ListOfTop(limit int) ([]ListOfTopItem, error) {

	rows, err := s.db.Query(context.Background(), `
		SELECT ip, SUM(count) AS c
		FROM ip_monitoring
		WHERE day_date = CURRENT_DATE
		GROUP BY ip
		ORDER BY c DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []ListOfTopItem{}

	for rows.Next() {
		var item ListOfTopItem
		if err := rows.Scan(&item.IP, &item.Count); err != nil {
			return nil, err
		}

		result = append(result, item)
	}

	return result, nil
}

// ListByBanProfile ListByBanProfile
func (s *Monitoring) ListByBanProfile(profile AutobanProfile) ([]net.IP, error) {
	group := append([]string{"ip"}, profile.Group...)

	rows, err := s.db.Query(context.Background(), `
		SELECT ip, SUM(count) AS c
		FROM ip_monitoring
		WHERE day_date = CURRENT_DATE
		GROUP BY `+strings.Join(group, ", ")+`
		HAVING SUM(count) > $1
		LIMIT 1000
	`, profile.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []net.IP{}

	for rows.Next() {
		var ip net.IP
		var c int
		if err := rows.Scan(&ip, &c); err != nil {
			return nil, err
		}

		result = append(result, ip)
	}

	return result, nil
}

// ExistsIP ban list already contains IP
func (s *Monitoring) ExistsIP(ip net.IP) (bool, error) {
	var exists bool
	err := s.db.QueryRow(context.Background(), `
		SELECT true
		FROM ip_monitoring
		WHERE ip = $1
		LIMIT 1
	`, ip.String()).Scan(&exists)
	if err != nil {
		if err != pgx.ErrNoRows {
			return false, err
		}

		return false, nil
	}

	return true, nil
}
