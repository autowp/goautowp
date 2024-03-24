package goautowp

import (
	"context"
	"crypto/sha1" //nolint:gosec
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/autowp/goautowp/itemofday"
	"github.com/gin-gonic/gin"
	decimal2 "github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type YoomoneyHandler struct {
	price              decimal2.Decimal
	notificationSecret string
	itemOfDay          *itemofday.Repository
}

func NewYoomoneyHandler(
	price string,
	notificationSecret string,
	itemOfDay *itemofday.Repository,
) (*YoomoneyHandler, error) {
	decPrice, err := decimal2.NewFromString(price)
	if err != nil {
		return nil, err
	}

	return &YoomoneyHandler{
		price:              decPrice,
		notificationSecret: notificationSecret,
		itemOfDay:          itemOfDay,
	}, nil
}

const RUB = "643"

type YoomoneyWebhook struct {
	NotificationType string `json:"notification_type" form:"notification_type" binding:"required"`
	OperationID      string `json:"operation_id"      form:"operation_id"      binding:"required"`
	Amount           string `json:"amount"            form:"amount"            binding:"required"`
	WithdrawAmount   string `json:"withdraw_amount"   form:"withdraw_amount"   binding:"required"`
	Currency         string `json:"currency"          form:"currency"          binding:"required"`
	Datetime         string `json:"datetime"          form:"datetime"          binding:"required"`
	Sender           string `json:"sender"            form:"sender"            binding:"required"`
	Codepro          string `json:"codepro"           form:"codepro"           binding:"required"`
	Label            string `json:"label"             form:"label"             binding:"required"`
	SHA1Hash         string `json:"sha1_hash"         form:"sha1_hash"         binding:"required"`
	TestNotification bool   `json:"test_notification" form:"test_notification"`
	Unaccepted       bool   `json:"unaccepted"        form:"unaccepted"`
}

func (s *YoomoneyHandler) Hash(fields YoomoneyWebhook) (string, error) {
	if s.notificationSecret == "" {
		return "", errors.New("notification secret not configured")
	}

	str := strings.Join([]string{
		fields.NotificationType,
		fields.OperationID,
		fields.Amount,
		fields.Currency,
		fields.Datetime,
		fields.Sender,
		fields.Codepro,
		s.notificationSecret,
		fields.Label,
	}, "&")

	h := sha1.New() //nolint:gosec
	h.Write([]byte(str))
	sha1hash := h.Sum(nil)

	return hex.EncodeToString(sha1hash[0:sha1.Size]), nil
}

func (s *YoomoneyHandler) Handle(ctx context.Context, fields YoomoneyWebhook) error {
	sha1hashStr, err := s.Hash(fields)
	if err != nil {
		return err
	}

	if fields.SHA1Hash == "" {
		return errors.New("sha1_hash not provided")
	}

	if !strings.EqualFold(sha1hashStr, fields.SHA1Hash) {
		return errors.New("sha1 hash not matched")
	}

	if fields.Currency != RUB {
		return errors.New("only RUB currency is supported")
	}

	withdrawAmount, err := decimal2.NewFromString(fields.WithdrawAmount)
	if err != nil {
		return err
	}

	if s.price.GreaterThan(withdrawAmount) {
		return errors.New("price is greater than withdraw_amount")
	}

	re := regexp.MustCompile(`^vod/(\d{4}-\d{2}-\d{2})/(\d+)/(\d+)$`)

	matches := re.FindStringSubmatch(fields.Label)
	if matches == nil || len(matches) < 2 {
		return errors.New("label not matched by regular expression")
	}

	dateTime, err := time.Parse("2006-01-02", matches[1])
	if err != nil {
		return fmt.Errorf("failed to parse date in label: %w", err)
	}

	IsAvailableDate, err := s.itemOfDay.IsAvailableDate(ctx, dateTime)
	if err != nil {
		return err
	}

	if !IsAvailableDate {
		return errors.New("date is not available")
	}

	itemID, err := strconv.ParseInt(matches[2], 10, 0)
	if err != nil {
		return err
	}

	userID, err := strconv.ParseInt(matches[3], 10, 0)
	if err != nil {
		return err
	}

	if itemID == 0 {
		return errors.New("item_id can't be 0")
	}

	if fields.Unaccepted {
		return errors.New("unaccepted payment")
	}

	success, err := s.itemOfDay.SetItemOfDay(ctx, dateTime, itemID, userID)
	if err != nil {
		return err
	}

	if !success {
		return errors.New("failed to set item of day")
	}

	return nil
}

func (s *YoomoneyHandler) SetupRouter(ctx context.Context, r *gin.Engine) {
	r.POST("/yoomoney/informing", func(c *gin.Context) {
		fields := YoomoneyWebhook{}

		err := c.ShouldBind(fields)
		if err != nil {
			logrus.Warn("yoomoney: bad request")
			c.Status(http.StatusBadRequest)

			return
		}

		err = s.Handle(ctx, fields)
		if err != nil {
			logrus.Warnf("yoomoney: %s", err.Error())
			c.String(http.StatusInternalServerError, err.Error())

			return
		}

		logrus.Info("yoomoney: success")
		c.Status(http.StatusOK)
	})
}
