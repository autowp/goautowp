package ban

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"

	"github.com/sirupsen/logrus"
)

var ErrBanItemNotFound = errors.New("ban item not found")

// Item Item.
type Item struct {
	IP       net.IP    `json:"ip"`
	Until    time.Time `json:"up_to"`
	ByUserID int64     `json:"by_user_id"`
	Reason   string    `json:"reason"`
}

// Repository Main Object.
type Repository struct {
	db *goqu.Database
}

// NewRepository constructor.
func NewRepository(db *goqu.Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	s := &Repository{
		db: db,
	}

	return s, nil
}

// Add IP to list of banned.
func (s *Repository) Add(ip net.IP, duration time.Duration, byUserID int64, reason string) error {
	reason = strings.TrimSpace(reason)
	upTo := time.Now().Add(duration)

	ct, err := s.db.ExecContext(context.Background(), `
		INSERT INTO ip_ban (ip, until, by_user_id, reason)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT(ip) DO UPDATE SET until = EXCLUDED.until, by_user_id = EXCLUDED.by_user_id, reason = EXCLUDED.reason
	`, ip.String(), upTo, byUserID, reason)
	if err != nil {
		return err
	}

	affected, err := ct.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 1 {
		logrus.Infof("%v was banned. Reason: %s", ip.String(), reason)
	}

	return nil
}

// Remove IP from list of banned.
func (s *Repository) Remove(ip net.IP) error {
	logrus.Info(ip.String() + ": unban")
	_, err := s.db.ExecContext(context.Background(), "DELETE FROM ip_ban WHERE ip = $1", ip.String())

	return err
}

// Exists ban list already contains IP.
func (s *Repository) Exists(ip net.IP) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(context.Background(), `
		SELECT true
		FROM ip_ban
		WHERE ip = $1 AND until >= NOW()
	`, ip.String()).Scan(&exists)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}

	return !errors.Is(err, sql.ErrNoRows), nil
}

// Get ban info.
func (s *Repository) Get(ip net.IP) (*Item, error) {
	item := Item{}

	err := s.db.QueryRowContext(context.Background(), `
		SELECT ip, until, reason, by_user_id
		FROM ip_ban
		WHERE ip = $1 AND until >= NOW()
	`, ip.String()).Scan(&item.IP, &item.Until, &item.Reason, &item.ByUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBanItemNotFound
		}

		return nil, err
	}

	return &item, nil
}

// GC Garbage Collect.
func (s *Repository) GC() (int64, error) {
	ct, err := s.db.ExecContext(context.Background(), "DELETE FROM ip_ban WHERE until < NOW()")
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
func (s *Repository) Clear() error {
	_, err := s.db.ExecContext(context.Background(), "DELETE FROM ip_ban")

	return err
}
