package goautowp

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"net"
	"strings"
	"time"
)

// BanItem BanItem
type BanItem struct {
	IP       net.IP    `json:"ip"`
	Until    time.Time `json:"up_to"`
	ByUserID int       `json:"by_user_id"`
	Reason   string    `json:"reason"`
}

// BanRepository Main Object
type BanRepository struct {
	db *pgxpool.Pool
}

// NewBanRepository constructor
func NewBanRepository(db *pgxpool.Pool) (*BanRepository, error) {

	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	s := &BanRepository{
		db: db,
	}

	return s, nil
}

// Add IP to list of banned
func (s *BanRepository) Add(ip net.IP, duration time.Duration, byUserID int64, reason string) error {
	reason = strings.TrimSpace(reason)
	upTo := time.Now().Add(duration)

	ct, err := s.db.Exec(context.Background(), `
		INSERT INTO ip_ban (ip, until, by_user_id, reason)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT(ip) DO UPDATE SET until=EXCLUDED.until, by_user_id=EXCLUDED.by_user_id, reason=EXCLUDED.reason
	`, ip.String(), upTo, byUserID, reason)
	if err != nil {
		return err
	}

	affected := ct.RowsAffected()

	if affected == 1 {
		log.Printf("%v was banned. Reason: %s\n", ip.String(), reason)
	}

	return nil
}

// Remove IP from list of banned
func (s *BanRepository) Remove(ip net.IP) error {
	log.Println(ip.String() + ": unban")
	_, err := s.db.Exec(context.Background(), "DELETE FROM ip_ban WHERE ip = $1", ip.String())

	return err
}

// Exists ban list already contains IP
func (s *BanRepository) Exists(ip net.IP) (bool, error) {

	var exists bool
	err := s.db.QueryRow(context.Background(), `
		SELECT true
		FROM ip_ban
		WHERE ip = $1 AND until >= NOW()
	`, ip.String()).Scan(&exists)
	if err != nil && err != pgx.ErrNoRows {
		return false, err
	}

	return err != pgx.ErrNoRows, nil
}

// Get ban info
func (s *BanRepository) Get(ip net.IP) (*BanItem, error) {

	item := BanItem{}
	err := s.db.QueryRow(context.Background(), `
		SELECT ip, until, reason, by_user_id
		FROM ip_ban
		WHERE ip = $1 AND until >= NOW()
	`, ip.String()).Scan(&item.IP, &item.Until, &item.Reason, &item.ByUserID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &item, nil
}

// GC Garbage Collect
func (s *BanRepository) GC() (int64, error) {
	ct, err := s.db.Exec(context.Background(), "DELETE FROM ip_ban WHERE until < NOW()")
	if err != nil {
		return 0, err
	}

	affected := ct.RowsAffected()

	return affected, nil
}

// Clear removes all collected data
func (s *BanRepository) Clear() error {
	_, err := s.db.Exec(context.Background(), "DELETE FROM ip_ban")

	return err
}
