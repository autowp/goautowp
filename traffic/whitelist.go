package traffic

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"net"
	"strings"

	"github.com/autowp/goautowp/util"

	"github.com/doug-martin/goqu/v9"

	"github.com/sirupsen/logrus"
)

var ErrWhitelistItemNotFound = errors.New("whitelist item not found")

// Whitelist Main Object.
type Whitelist struct {
	db *goqu.Database
}

// WhitelistItem WhitelistItem.
type WhitelistItem struct {
	IP          net.IP `json:"ip"`
	Description string `json:"description"`
}

// NewWhitelist constructor.
func NewWhitelist(db *goqu.Database) (*Whitelist, error) {
	return &Whitelist{
		db: db,
	}, nil
}

// MatchAuto MatchAuto.
func (s *Whitelist) MatchAuto(ip net.IP) (bool, string) {
	ipText := ip.String()
	ipWithDashes := strings.ReplaceAll(ipText, ".", "-")

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

// Add IP to whitelist.
func (s *Whitelist) Add(ctx context.Context, ip net.IP, desc string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO ip_whitelist (ip, description)
		VALUES ($1, $2)
		ON CONFLICT (ip) DO UPDATE SET description=EXCLUDED.description
	`, ip.String(), desc)

	return err
}

// Get whitelist item.
func (s *Whitelist) Get(ctx context.Context, ip net.IP) (*WhitelistItem, error) {
	var item WhitelistItem

	err := s.db.QueryRowContext(ctx, `
		SELECT ip, description
		FROM ip_whitelist
		WHERE ip = $1
	`, ip.String()).Scan(&item.IP, item.Description)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWhitelistItemNotFound
		}

		return nil, err
	}

	return &item, nil
}

// List whitelist items.
func (s *Whitelist) List(ctx context.Context) ([]*WhitelistItem, error) {
	result := make([]*WhitelistItem, 0)

	rows, err := s.db.QueryContext(ctx, `
		SELECT ip, description
		FROM ip_whitelist
	`)
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	for rows.Next() {
		var item WhitelistItem
		if err := rows.Scan(&item.IP, &item.Description); err != nil {
			return nil, err
		}

		result = append(result, &item)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// Exists whitelist already contains IP.
func (s *Whitelist) Exists(ctx context.Context, ip net.IP) (bool, error) {
	var exists bool

	err := s.db.QueryRowContext(ctx, `
		SELECT true
		FROM ip_whitelist
		WHERE ip = $1
	`, ip.String()).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// Remove IP from whitelist.
func (s *Whitelist) Remove(ctx context.Context, ip net.IP) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM ip_whitelist WHERE ip = $1", ip.String())

	return err
}
