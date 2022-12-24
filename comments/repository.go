package comments

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"

	"github.com/autowp/goautowp/items"

	"github.com/autowp/goautowp/hosts"

	"github.com/autowp/goautowp/messaging"

	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
)

type CommentType int32

const (
	TypeIDPictures CommentType = 1
	TypeIDItems    CommentType = 2
	TypeIDVotings  CommentType = 3
	TypeIDArticles CommentType = 4
	TypeIDForums   CommentType = 5
)

type ModeratorAttention int32

const (
	ModeratorAttentionNone      ModeratorAttention = 0
	ModeratorAttentionRequired  ModeratorAttention = 1
	ModeratorAttentionCompleted ModeratorAttention = 2
)

type GetVotesResult struct {
	PositiveVotes []users.DBUser
	NegativeVotes []users.DBUser
}

// Repository Main Object.
type Repository struct {
	db                *goqu.Database
	userRepository    *users.Repository
	messageRepository *messaging.Repository
	hostManager       *hosts.Manager
}

// NewRepository constructor.
func NewRepository(
	db *goqu.Database,
	userRepository *users.Repository,
	messageRepository *messaging.Repository,
	hostManager *hosts.Manager,
) *Repository {
	return &Repository{
		db:                db,
		userRepository:    userRepository,
		messageRepository: messageRepository,
		hostManager:       hostManager,
	}
}

func (s *Repository) GetVotes(ctx context.Context, id int64) (*GetVotesResult, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT users.id, users.name, users.deleted, users.identity, users.last_online, users.role, 
            users.specs_weight, comment_vote.vote
		FROM comment_vote
			INNER JOIN users ON comment_vote.user_id = users.id
		WHERE comment_vote.comment_id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	positiveVotes := make([]users.DBUser, 0)
	negativeVotes := make([]users.DBUser, 0)

	for rows.Next() {
		var (
			r    users.DBUser
			vote int
		)

		err = rows.Scan(&r.ID, &r.Name, &r.Deleted, &r.Identity, &r.LastOnline, &r.Role, &r.SpecsWeight, &vote)
		if err != nil {
			return nil, err
		}

		if vote > 0 {
			positiveVotes = append(positiveVotes, r)
		} else {
			negativeVotes = append(negativeVotes, r)
		}
	}

	return &GetVotesResult{
		PositiveVotes: positiveVotes,
		NegativeVotes: negativeVotes,
	}, nil
}

func (s *Repository) Subscribe(ctx context.Context, userID int64, commentsType CommentType, itemID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT IGNORE INTO comment_topic_subscribe (type_id, item_id, user_id, sent)
		VALUES (?, ?, ?, 0)
    `, commentsType, itemID, userID)

	return err
}

func (s *Repository) UnSubscribe(ctx context.Context, userID int64, commentsType CommentType, itemID int64) error {
	_, err := s.db.ExecContext(
		ctx,
		"DELETE FROM comment_topic_subscribe WHERE type_id = ? AND item_id = ? AND user_id = ?",
		commentsType, itemID, userID,
	)

	return err
}

func (s *Repository) View(ctx context.Context, userID int64, commentsType CommentType, itemID int64) error {
	_, err := s.db.ExecContext(
		ctx,
		`
			INSERT INTO comment_topic_view (user_id, type_id, item_id, timestamp)
            VALUES (?, ?, ?, NOW())
            ON DUPLICATE KEY UPDATE timestamp = values(timestamp)
        `,
		userID, commentsType, itemID,
	)

	return err
}

func (s *Repository) QueueDeleteMessage(ctx context.Context, commentID int64, byUserID int64) error {
	var moderatorAttention ModeratorAttention

	err := s.db.QueryRowContext(ctx, "SELECT moderator_attention FROM comment_message WHERE id = ?", commentID).
		Scan(&moderatorAttention)
	if err != nil {
		return err
	}

	if moderatorAttention == ModeratorAttentionRequired {
		return errors.New("comment with moderation attention requirement can't be deleted")
	}

	_, err = s.db.ExecContext(
		ctx,
		`
			UPDATE comment_message SET deleted = 1, deleted_by = ?, delete_date = NOW()
            WHERE id = ?
        `,
		byUserID, commentID,
	)

	return err
}

func (s *Repository) RestoreMessage(ctx context.Context, commentID int64) error {
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE comment_message SET deleted = 0, delete_date = null WHERE id = ?",
		commentID,
	)

	return err
}

func (s *Repository) GetCommentType(ctx context.Context, commentID int64) (CommentType, error) {
	var commentType CommentType
	err := s.db.QueryRowContext(ctx, "SELECT type_id FROM comment_message WHERE id = ?", commentID).Scan(&commentType)

	return commentType, err
}

func (s *Repository) MoveMessage(ctx context.Context, commentID int64, dstType CommentType, dstItemID int64) error {
	var (
		srcType   CommentType
		srcItemID int64
	)

	err := s.db.QueryRowContext(ctx, "SELECT type_id, item_id FROM comment_message WHERE id = ?", commentID).
		Scan(&srcType, &srcItemID)
	if err != nil {
		return err
	}

	if srcItemID == dstItemID && srcType == dstType {
		return nil
	}

	_, err = s.db.ExecContext(
		ctx,
		"UPDATE comment_message SET type_id = ?, item_id = ?, parent_id = null WHERE id = ?",
		dstType, dstItemID, commentID,
	)
	if err != nil {
		return err
	}

	err = s.moveMessageRecursive(ctx, commentID, dstType, dstItemID)
	if err != nil {
		return err
	}

	err = s.updateTopicStat(ctx, srcType, srcItemID)
	if err != nil {
		return err
	}

	return s.updateTopicStat(ctx, dstType, dstItemID)
}

func (s *Repository) moveMessageRecursive(
	ctx context.Context,
	parentID int64,
	dstType CommentType,
	dstItemID int64,
) error {
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE comment_message SET type_id = ?, item_id = ? WHERE id = ?",
		dstType, dstItemID, parentID,
	)
	if err != nil {
		return err
	}

	rows, err := s.db.QueryContext(ctx, "SELECT id FROM comment_message WHERE parent_id = ?", parentID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	for rows.Next() {
		var id int64
		err = rows.Scan(&id)

		if err != nil {
			return err
		}

		err = s.moveMessageRecursive(ctx, id, dstType, dstItemID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) updateTopicStat(ctx context.Context, commentType CommentType, itemID int64) error {
	var (
		messagesCount int
		lastUpdate    *string
	)

	err := s.db.QueryRowContext(
		ctx,
		"SELECT COUNT(1), MAX(datetime) FROM comment_message WHERE type_id = ? AND item_id = ?",
		commentType, itemID,
	).Scan(&messagesCount, &lastUpdate)
	if err != nil {
		return err
	}

	if messagesCount <= 0 {
		_, err = s.db.ExecContext(
			ctx,
			"DELETE FROM comment_topic WHERE type_id = ? AND item_id = ?",
			commentType, itemID,
		)

		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		`
            INSERT INTO comment_topic (item_id, type_id, last_update, messages)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE last_update = VALUES(last_update), messages = VALUES(messages)
        `,
		itemID, commentType, lastUpdate, messagesCount,
	)

	return err
}

func (s *Repository) VoteComment(ctx context.Context, userID int64, commentID int64, vote int32) (int32, error) {
	if vote > 0 {
		vote = 1
	} else {
		vote = -1
	}

	var authorID int64

	err := s.db.QueryRowContext(
		ctx, "SELECT author_id FROM comment_message WHERE id = ?", commentID,
	).Scan(&authorID)
	if err != nil {
		return 0, err
	}

	if authorID == userID {
		return 0, errors.New("self-vote forbidden")
	}

	res, err := s.db.ExecContext(
		ctx,
		`
            INSERT INTO comment_vote (comment_id, user_id, vote)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE vote = VALUES(vote)
        `,
		commentID, userID, vote,
	)
	if err != nil {
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	if affected == 0 {
		return 0, errors.New("already voted")
	}

	newVote, err := s.updateVote(ctx, commentID)
	if err != nil {
		return 0, err
	}

	return newVote, nil
}

func (s *Repository) updateVote(ctx context.Context, commentID int64) (int32, error) {
	var count int32

	err := s.db.QueryRowContext(
		ctx,
		"SELECT sum(vote) FROM comment_vote WHERE comment_id = ?",
		commentID,
	).Scan(&count)
	if err != nil {
		return 0, err
	}

	_, err = s.db.ExecContext(
		ctx, "UPDATE comment_message SET vote = ? WHERE id = ?", count, commentID,
	)

	return count, err
}

func (s *Repository) CompleteMessage(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(
		ctx, "UPDATE comment_message SET moderator_attention = ? WHERE id = ? AND moderator_attention = ?",
		ModeratorAttentionCompleted, id, ModeratorAttentionRequired,
	)

	return err
}

func (s *Repository) Add(
	ctx context.Context,
	typeID CommentType,
	itemID int64,
	parentID int64,
	userID int64,
	message string,
	addr string,
	attention bool,
) (int64, error) {
	if parentID > 0 {
		deleted := false
		err := s.db.QueryRowContext(
			ctx,
			"SELECT deleted FROM comment_message WHERE type_id = ? AND item_id = ? AND id = ?",
			typeID, itemID, parentID,
		).Scan(&deleted)

		if err == sql.ErrNoRows {
			return 0, errors.New("message not found")
		} else if err != nil {
			return 0, err
		}

		if deleted {
			return 0, errors.New("message is deleted")
		}
	}

	ma := ModeratorAttentionNone
	if attention {
		ma = ModeratorAttentionRequired
	}

	res, err := s.db.Insert("comment_message").
		Cols("datetime", "type_id", "item_id", "parent_id", "author_id", "message", "ip", "moderator_attention").
		Vals(goqu.Vals{
			goqu.L("NOW()"),
			typeID,
			itemID,
			sql.NullInt64{
				Int64: parentID,
				Valid: parentID > 0,
			},
			userID,
			message,
			goqu.L("INET6_ATON(?)", addr),
			ma,
		}).Executor().ExecContext(ctx)
	if err != nil {
		return 0, err
	}

	messageID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if parentID > 0 {
		err = s.UpdateMessageRepliesCount(ctx, parentID)
		if err != nil {
			return 0, err
		}
	}

	err = s.UpdateTopicStat(ctx, typeID, itemID)
	if err != nil {
		return 0, err
	}

	err = s.UpdateTopicView(ctx, typeID, itemID, userID)
	if err != nil {
		return 0, err
	}

	return messageID, nil
}

func (s *Repository) UpdateMessageRepliesCount(ctx context.Context, messageID int64) error {
	var count int64

	err := s.db.QueryRowContext(
		ctx,
		"SELECT count(1) FROM comment_message WHERE parent_id = ?",
		messageID,
	).Scan(&count)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(
		ctx, "UPDATE comment_message SET replies_count = ? WHERE id = ?", count, messageID,
	)

	return err
}

func (s *Repository) UpdateTopicStat(ctx context.Context, typeID CommentType, itemID int64) error {
	messagesCount, err := s.countMessages(ctx, typeID, itemID)
	if err != nil {
		return err
	}

	if messagesCount <= 0 {
		_, err = s.db.ExecContext(
			ctx, "DELETE FROM comment_topic WHERE item_id = ? AND type_id = ?", itemID, typeID,
		)

		return err
	}

	lastUpdate, err := s.getLastUpdate(ctx, typeID, itemID)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(
		ctx, `
			INSERT INTO comment_topic (item_id, type_id, last_update, messages)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE last_update = VALUES(last_update), messages = VALUES(messages)
		`,
		itemID, typeID, lastUpdate, messagesCount,
	)

	return err
}

func (s *Repository) countMessages(ctx context.Context, typeID CommentType, itemID int64) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(
		ctx,
		"SELECT count(1) FROM comment_message WHERE item_id = ? AND type_id = ?",
		typeID, itemID,
	).Scan(&count)

	return count, err
}

func (s *Repository) getLastUpdate(ctx context.Context, typeID CommentType, itemID int64) (sql.NullTime, error) {
	var t sql.NullTime
	err := s.db.QueryRowContext(
		ctx,
		"SELECT datetime FROM comment_message WHERE item_id = ? AND type_id = ? ORDER BY datetime DESC LIMIT 1",
		itemID, typeID,
	).Scan(&t)

	if err == sql.ErrNoRows {
		err = nil
		t.Valid = false
	}

	return t, err
}

func (s *Repository) UpdateTopicView(ctx context.Context, typeID CommentType, itemID int64, userID int64) error {
	_, err := s.db.ExecContext(
		ctx, `
			INSERT INTO comment_topic_view (user_id, type_id, item_id, timestamp)
			VALUES (?, ?, ?, NOW())
			ON DUPLICATE KEY UPDATE timestamp = VALUES(timestamp)
		`,
		userID, typeID, itemID,
	)

	return err
}

func (s *Repository) AssertItem(ctx context.Context, typeID CommentType, itemID int64) error {
	var err error

	switch typeID {
	case TypeIDPictures:
		err = s.db.QueryRowContext(ctx, "SELECT 1 FROM pictures WHERE id = ?", itemID).Scan()

	case TypeIDItems:
		err = s.db.QueryRowContext(ctx, "SELECT 1 FROM item WHERE id = ?", itemID).Scan()

	case TypeIDVotings:
		err = s.db.QueryRowContext(ctx, "SELECT 1 FROM voting WHERE id = ?", itemID).Scan()

	case TypeIDArticles:
		err = s.db.QueryRowContext(ctx, "SELECT 1 FROM articles WHERE id = ?", itemID).Scan()

	case TypeIDForums:
		err = s.db.QueryRowContext(ctx, "SELECT 1 FROM forums_topics WHERE id = ?", itemID).Scan()

	default:
		err = errors.New("invalid type")
	}

	return err
}

func (s *Repository) NotifySubscribers(ctx context.Context, messageID int64) error {
	var (
		itemID, typeID int64
		authorID       sql.NullInt64
		authorIdentity string
	)

	err := s.db.QueryRowContext(
		ctx,
		"SELECT item_id, type_id, author_id FROM comment_message WHERE id = ?",
		messageID,
	).Scan(&itemID, &typeID, &authorID)
	if err != nil {
		return err
	}

	if !authorID.Valid {
		return nil
	}

	err = s.db.QueryRowContext(ctx, "SELECT identity FROM users WHERE id = ?", authorID.Int64).Scan(&authorIdentity)
	if err != nil {
		return err
	}

	ids, err := s.getSubscribersIDs(ctx, typeID, itemID, true)
	if err != nil {
		return err
	}

	filteredIDs := make([]int64, 0)

	for _, id := range ids {
		prefs, err := s.userRepository.UserPreferences(ctx, id, authorID.Int64)
		if err != nil {
			return err
		}

		if !prefs.DisableCommentsNotifications {
			filteredIDs = append(filteredIDs, id)
		}
	}

	if len(filteredIDs) == 0 {
		return nil
	}

	subscribers, err := s.db.QueryContext(
		ctx,
		"SELECT id, language FROM users WHERE id IN (?) and id != ?",
		filteredIDs, authorID,
	)
	if err != nil {
		return err
	}

	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	_, err = bundle.LoadMessageFile("en.json")
	if err != nil {
		return err
	}

	var (
		subscriberID       int64
		subscriberLanguage string
	)

	for subscribers.Next() {
		err = subscribers.Scan(&subscriberID, &subscriberLanguage)
		if err != nil {
			return err
		}

		path := "user" + strconv.FormatInt(authorID.Int64, 10)
		if len(authorIdentity) > 0 {
			path = authorIdentity
		}

		uri, err := s.hostManager.GetURIByLanguage(subscriberLanguage)
		if err != nil {
			return err
		}

		uri.Path = "/users/" + path
		userURL := uri.String()

		messageURL, err := s.getMessageURL(ctx, messageID, uri)
		if err != nil {
			return err
		}

		localizer := i18n.NewLocalizer(bundle, subscriberLanguage)

		message, err := localizer.Localize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID: "pm/user-%s-post-new-message-%s",
			},
			TemplateData: map[string]interface{}{
				"Name":    userURL,
				"Message": messageURL,
			},
		})
		if err != nil {
			return err
		}

		err = s.messageRepository.CreateMessage(ctx, 0, subscriberID, message)
		if err != nil {
			return err
		}

		err = s.setSubscriptionSent(ctx, typeID, itemID, subscriberID, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) getSubscribersIDs(
	ctx context.Context,
	typeID int64,
	itemID int64,
	onlyAwaiting bool,
) ([]int64, error) {
	sel := s.db.Select("user_id").From("comment_topic_subscribe").Where(
		goqu.I("type_id").Eq(typeID),
		goqu.I("item_id").Eq(itemID),
	)

	if onlyAwaiting {
		sel = sel.Where(goqu.L("NOT sent"))
	}

	result := make([]int64, 0)

	err := sel.Executor().ScanValsContext(ctx, &result)

	return result, err
}

func (s *Repository) setSubscriptionSent(
	ctx context.Context,
	typeID int64,
	itemID int64,
	subscriberID int64,
	sent bool,
) error {
	_, err := s.db.Update("comment_topic_subscribe").
		Set(goqu.Record{"sent": sent}).
		Where(
			goqu.I("type_id").Eq(typeID),
			goqu.I("item_id").Eq(itemID),
			goqu.I("user_id").Eq(subscriberID),
		).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) getMessageURL(ctx context.Context, messageID int64, uri *url.URL) (string, error) {
	var (
		itemID int64
		typeID CommentType
	)

	err := s.db.QueryRowContext(
		ctx,
		"SELECT item_id, type_id FROM comment_message WHERE id = ?",
		messageID,
	).Scan(&itemID, &typeID)
	if err != nil {
		return "", err
	}

	route, err := s.getMessageRowRoute(ctx, typeID, itemID)
	if err != nil {
		return "", err
	}

	route[0] = strings.TrimLeft(route[0], "/")

	for idx, val := range route {
		route[idx] = url.QueryEscape(val)
	}

	uri.Path = "/" + strings.Join(route, "/")
	uri.Fragment = "msg" + strconv.FormatInt(messageID, 10)

	return uri.String(), nil
}

func (s *Repository) getMessageRowRoute(ctx context.Context, typeID CommentType, itemID int64) ([]string, error) {
	switch typeID {
	case TypeIDPictures:
		var identity string

		err := s.db.QueryRowContext(ctx, "SELECT identity FROM pictures WHERE id = ?", itemID).Scan(&identity)
		if err != nil {
			return nil, err
		}

		return []string{"/picture", identity}, nil

	case TypeIDItems:
		var itemTypeID items.ItemType

		err := s.db.QueryRowContext(ctx, "SELECT item_type_id FROM item WHERE id = ?", itemID).Scan(&itemTypeID)
		if err != nil {
			return nil, err
		}

		switch itemTypeID {
		case items.TWINS:
			return []string{"/twins", "group", strconv.FormatInt(itemID, 10)}, nil
		case items.MUSEUM:
			return []string{"/museums", strconv.FormatInt(itemID, 10)}, nil
		default:
			return nil, fmt.Errorf(
				"failed to build url form message `%v` item_type `%v`",
				itemID,
				itemTypeID,
			)
		}

	case TypeIDVotings:
		return []string{"/voting", strconv.FormatInt(itemID, 10)}, nil

	case TypeIDArticles:
		var catname string

		err := s.db.QueryRowContext(ctx, "SELECT catname FROM articles WHERE id = ?", itemID).Scan(&catname)
		if err != nil {
			return nil, err
		}

		return []string{"/articles", catname}, nil

	case TypeIDForums:
		return []string{"/forums", "message", strconv.FormatInt(itemID, 10)}, nil
	}

	return nil, fmt.Errorf("unknown type_id `%v`", typeID)
}
