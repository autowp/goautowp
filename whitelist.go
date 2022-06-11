package goautowp

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
	"net"
	"strings"
)

// Whitelist Main Object
type Whitelist struct {
	db *pgxpool.Pool
}

// WhitelistItem WhitelistItem
type WhitelistItem struct {
	IP          net.IP `json:"ip"`
	Description string `json:"description"`
}

// NewWhitelist constructor
func NewWhitelist(db *pgxpool.Pool) (*Whitelist, error) {
	return &Whitelist{
		db: db,
	}, nil
}

// MatchAuto MatchAuto
func (s *Whitelist) MatchAuto(ip net.IP) (bool, string) {
	ipText := ip.String()
	ipWithDashes := strings.Replace(ipText, ".", "-", -1)

	msnHost := "msnbot-" + ipWithDashes + ".search.msn.com."
	yandexComHost := ipWithDashes + ".spider.yandex.com."
	googlebotHost := "crawl-" + ipWithDashes + ".googlebot.com."

	isIPv6 := len(ip) == net.IPv6len
	ip16 := ip.To16()

	yandexComIPv6Host := hex.EncodeToString(ip16[12:14]) + "-" + hex.EncodeToString(ip16[14:16]) + ".spider.yandex.com."

	hosts, err := net.LookupAddr(ipText)
	if err != nil {
		return false, ""
	}

	for _, host := range hosts {
		logrus.Info(host + " ")

		if host == msnHost {
			return true, "msnbot autodetect"
		}

		if host == yandexComHost {
			return true, "yandex.com autodetect"
		}

		if host == googlebotHost {
			return true, "googlebot autodetect"
		}

		if isIPv6 && host == yandexComIPv6Host {
			return true, "yandex.com ipv6 autodetect"
		}
	}

	return false, ""
}

// Add IP to whitelist
func (s *Whitelist) Add(ip net.IP, desc string) error {
	_, err := s.db.Exec(context.Background(), `
		INSERT INTO ip_whitelist (ip, description)
		VALUES ($1, $2)
		ON CONFLICT (ip) DO UPDATE SET description=EXCLUDED.description
	`, ip.String(), desc)

	return err
}

// Get whitelist item
func (s *Whitelist) Get(ip net.IP) (*WhitelistItem, error) {
	var item WhitelistItem
	err := s.db.QueryRow(context.Background(), `
		SELECT ip, description
		FROM ip_whitelist
		WHERE ip = $1
	`, ip.String()).Scan(&item.IP, item.Description)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &item, nil
}

// List whitelist items
func (s *Whitelist) List() ([]*APITrafficWhitelistItem, error) {
	result := make([]*APITrafficWhitelistItem, 0)
	rows, err := s.db.Query(context.Background(), `
		SELECT ip, description
		FROM ip_whitelist
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item APITrafficWhitelistItem
		if err := rows.Scan(&item.Ip, &item.Description); err != nil {
			return nil, err
		}

		result = append(result, &item)
	}

	return result, nil
}

// Exists whitelist already contains IP
func (s *Whitelist) Exists(ip net.IP) (bool, error) {
	var exists bool
	err := s.db.QueryRow(context.Background(), `
		SELECT true
		FROM ip_whitelist
		WHERE ip = $1
	`, ip.String()).Scan(&exists)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// Remove IP from whitelist
func (s *Whitelist) Remove(ip net.IP) error {
	_, err := s.db.Exec(context.Background(), "DELETE FROM ip_whitelist WHERE ip = $1", ip.String())

	return err
}
