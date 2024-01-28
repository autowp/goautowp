package traffic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgtype"
	"github.com/sirupsen/logrus"
)

// Monitoring Main Object.
type Monitoring struct {
	db *goqu.Database
}

// MonitoringInputMessage InputMessage.
type MonitoringInputMessage struct {
	IP        net.IP    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}

// ListOfTopItem ListOfTopItem.
type ListOfTopItem struct {
	IP    net.IP `json:"ip"`
	Count int    `json:"count"`
}

// NewMonitoring constructor.
func NewMonitoring(db *goqu.Database) (*Monitoring, error) {
	s := &Monitoring{
		db: db,
	}

	return s, nil
}

// Listen for incoming messages.
func (s *Monitoring) Listen(ctx context.Context, url string, queue string, quitChan chan bool) error {
	conn, err := util.ConnectRabbitMQ(url)
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

			err = s.Add(ctx, message.IP, message.Timestamp)
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

// Add item to Monitoring.
func (s *Monitoring) Add(ctx context.Context, ip net.IP, timestamp time.Time) error {
	_, err := s.db.ExecContext(ctx, `
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

// GC Garbage Collect.
func (s *Monitoring) GC(ctx context.Context) (int64, error) {
	ct, err := s.db.ExecContext(ctx, "DELETE FROM ip_monitoring WHERE day_date < CURRENT_DATE")
	if err != nil {
		return 0, err
	}

	affected, err := ct.RowsAffected()
	if err != nil {
		return 0, err
	}

	return affected, nil
}

// Clear removes all collected data.
func (s *Monitoring) Clear(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM ip_monitoring")

	return err
}

// ClearIP removes all data collected for IP.
func (s *Monitoring) ClearIP(ctx context.Context, ip net.IP) error {
	logrus.Info(ip.String() + ": clear monitoring")
	_, err := s.db.ExecContext(ctx, "DELETE FROM ip_monitoring WHERE ip = $1", ip.String())

	return err
}

// ListOfTop ListOfTop.
func (s *Monitoring) ListOfTop(ctx context.Context, limit int) ([]ListOfTopItem, error) {
	rows, err := s.db.QueryContext(ctx, `
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
	defer util.Close(rows)

	result := []ListOfTopItem{}

	for rows.Next() {
		var (
			ip   pgtype.Inet
			item ListOfTopItem
		)

		if err := rows.Scan(&ip, &item.Count); err != nil {
			return nil, err
		}

		if ip.IPNet != nil {
			item.IP = ip.IPNet.IP
		}

		result = append(result, item)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// ListByBanProfile ListByBanProfile.
func (s *Monitoring) ListByBanProfile(ctx context.Context, profile AutobanProfile) ([]net.IP, error) {
	group := append([]string{"ip"}, profile.Group...)

	rows, err := s.db.QueryContext(ctx, `
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
	defer util.Close(rows)

	result := []net.IP{}

	for rows.Next() {
		var (
			ip pgtype.Inet
			c  int
		)

		if err := rows.Scan(&ip, &c); err != nil {
			return nil, err
		}

		if ip.IPNet != nil {
			result = append(result, ip.IPNet.IP)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// ExistsIP ban list already contains IP.
func (s *Monitoring) ExistsIP(ip net.IP) (bool, error) {
	var exists bool

	err := s.db.QueryRowContext(context.Background(), `
		SELECT true
		FROM ip_monitoring
		WHERE ip = $1
		LIMIT 1
	`, ip.String()).Scan(&exists)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return false, err
		}

		return false, nil
	}

	return true, nil
}
