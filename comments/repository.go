package comments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/messaging"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const CommentMessagePreviewLength = 60

type CommentType int32

const (
	TypeIDPictures CommentType = 1
	TypeIDItems    CommentType = 2
	TypeIDVotings  CommentType = 3
	TypeIDArticles CommentType = 4
	TypeIDForums   CommentType = 5
)

const (
	tableCommentMessage   = "comment_message"
	colModeratorAttention = "moderator_attention"
)

const deleteTTLDays = 300

type ModeratorAttention int32

const (
	ModeratorAttentionNone      ModeratorAttention = 0
	ModeratorAttentionRequired  ModeratorAttention = 1
	ModeratorAttentionCompleted ModeratorAttention = 2
)

const MaxMessageLength = 16 * 1024

type GetVotesResult struct {
	PositiveVotes []users.DBUser
	NegativeVotes []users.DBUser
}

type CommentMessage struct {
	ID                 int64              `db:"id"`
	TypeID             CommentType        `db:"type_id"`
	ItemID             int64              `db:"item_id"`
	ParentID           sql.NullInt64      `db:"parent_id"`
	CreatedAt          time.Time          `db:"datetime"`
	Deleted            bool               `db:"deleted"`
	ModeratorAttention ModeratorAttention `db:"moderator_attention"`
	AuthorID           sql.NullInt64      `db:"author_id"`
	IP                 net.IP             `db:"ip"`
	Message            string             `db:"message"`
	Vote               int32              `db:"vote"`
}

type Request struct {
	ItemID             int64
	TypeID             CommentType
	ParentID           int64
	NoParents          bool
	UserID             int64
	Order              []exp.OrderedExpression
	ModeratorAttention ModeratorAttention
	PicturesOfItemID   int64
	FetchMessage       bool
	FetchVote          bool
	FetchIP            bool
	PerPage            int32
	Page               int32
}

// Repository Main Object.
type Repository struct {
	db                *goqu.Database
	userRepository    *users.Repository
	messageRepository *messaging.Repository
	hostManager       *hosts.Manager
	i18n              *i18nbundle.I18n
}

// NewRepository constructor.
func NewRepository(
	db *goqu.Database,
	userRepository *users.Repository,
	messageRepository *messaging.Repository,
	hostManager *hosts.Manager,
	i18n *i18nbundle.I18n,
) *Repository {
	return &Repository{
		db:                db,
		userRepository:    userRepository,
		messageRepository: messageRepository,
		hostManager:       hostManager,
		i18n:              i18n,
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

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &GetVotesResult{
		PositiveVotes: positiveVotes,
		NegativeVotes: negativeVotes,
	}, nil
}

func (s *Repository) IsSubscribed(
	ctx context.Context, userID int64, commentsType CommentType, itemID int64,
) (bool, error) {
	var result bool
	success, err := s.db.ScanValContext(ctx, &result, `
		SELECT 1 FROM comment_topic_subscribe WHERE type_id = ? AND item_id = ? AND user_id = ?
    `, commentsType, itemID, userID)

	return success && result, err
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

	return rows.Err()
}

func (s *Repository) updateTopicStat(ctx context.Context, commentType CommentType, itemID int64) error {
	var (
		messagesCount int
		lastUpdate    *sql.NullTime
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

	if lastUpdate.Valid {
		_, err = s.db.ExecContext(
			ctx,
			`
				INSERT INTO comment_topic (item_id, type_id, last_update, messages)
				VALUES (?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE last_update = VALUES(last_update), messages = VALUES(messages)
			`,
			itemID, commentType, lastUpdate.Time.Format("2006-01-02 15:04:05"), messagesCount,
		)
	}

	return err
}

func (s *Repository) UserVote(ctx context.Context, userID int64, commentID int64) (int32, error) {
	var vote int32
	err := s.db.QueryRowContext(
		ctx, "SELECT vote FROM comment_vote WHERE comment_id = ? AND user_id = ?", commentID, userID,
	).Scan(&vote)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}

	return vote, err
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

		if errors.Is(err, sql.ErrNoRows) {
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

	res, err := s.db.Insert(tableCommentMessage).
		Cols("datetime", "type_id", "item_id", "parent_id", "author_id", "message", "ip", colModeratorAttention).
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

	err = s.updateTopicStat(ctx, typeID, itemID)
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
	var (
		err error
		val int
	)

	switch typeID {
	case TypeIDPictures:
		err = s.db.QueryRowContext(ctx, "SELECT 1 FROM pictures WHERE id = ?", itemID).Scan(&val)

	case TypeIDItems:
		err = s.db.QueryRowContext(ctx, "SELECT 1 FROM item WHERE id = ?", itemID).Scan(&val)

	case TypeIDVotings:
		err = s.db.QueryRowContext(ctx, "SELECT 1 FROM voting WHERE id = ?", itemID).Scan(&val)

	case TypeIDArticles:
		err = s.db.QueryRowContext(ctx, "SELECT 1 FROM articles WHERE id = ?", itemID).Scan(&val)

	case TypeIDForums:
		err = s.db.QueryRowContext(ctx, "SELECT 1 FROM forums_topics WHERE id = ?", itemID).Scan(&val)

	default:
		err = errors.New("invalid type")
	}

	return err
}

func (s *Repository) NotifyAboutReply(ctx context.Context, messageID int64) error {
	var (
		authorID       int64
		parentAuthorID int64
		authorIdentity sql.NullString
		parentLanguage string
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT cm1.author_id, parent_message.author_id, u.identity, parent_user.language
		FROM comment_message AS cm1 
		    JOIN comment_message AS parent_message ON cm1.parent_id = parent_message.id
			JOIN users AS u ON cm1.author_id = u.id
			JOIN users AS parent_user ON parent_message.author_id = parent_user.id
		WHERE cm1.id = ? AND cm1.author_id != parent_message.author_id AND NOT parent_user.deleted
    `, messageID).Scan(&authorID, &parentAuthorID, &authorIdentity, &parentLanguage)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return err
	}

	ai := ""
	if authorIdentity.Valid {
		ai = authorIdentity.String
	}

	userURL, err := s.userURL(authorID, ai, parentLanguage)
	if err != nil {
		return err
	}

	uri, err := s.hostManager.URIByLanguage(parentLanguage)
	if err != nil {
		return err
	}

	messageURL, err := s.messageURL(ctx, messageID, uri)
	if err != nil {
		return err
	}

	localizer := s.i18n.Localizer(parentLanguage)

	message, err := localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: "pm/user-%s-replies-to-you-%s",
		},
		TemplateData: map[string]interface{}{
			"Name":    userURL,
			"Message": messageURL,
		},
	})
	if err != nil {
		return err
	}

	return s.messageRepository.CreateMessage(ctx, 0, parentAuthorID, message)
}

func (s *Repository) NotifySubscribers(ctx context.Context, messageID int64) error {
	var (
		itemID         int64
		typeID         CommentType
		authorID       sql.NullInt64
		authorIdentity sql.NullString
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

	au := ""
	if authorIdentity.Valid {
		au = authorIdentity.String
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

	subscribers, err := s.db.From("users").Select("id", "language").Where(
		goqu.I("id").In(filteredIDs),
		goqu.I("id").Neq(authorID),
	).Executor().QueryContext(ctx)
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

		userURL, err := s.userURL(authorID.Int64, au, subscriberLanguage)
		if err != nil {
			return err
		}

		uri, err := s.hostManager.URIByLanguage(subscriberLanguage)
		if err != nil {
			return err
		}

		messageURL, err := s.messageURL(ctx, messageID, uri)
		if err != nil {
			return err
		}

		localizer := s.i18n.Localizer(subscriberLanguage)

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

		err = s.SetSubscriptionSent(ctx, typeID, itemID, subscriberID, true)
		if err != nil {
			return err
		}
	}

	return subscribers.Err()
}

func (s *Repository) getSubscribersIDs(
	ctx context.Context,
	typeID CommentType,
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

func (s *Repository) SetSubscriptionSent(
	ctx context.Context,
	typeID CommentType,
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

func (s *Repository) messageURL(ctx context.Context, messageID int64, uri *url.URL) (string, error) {
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

	route, err := s.messageRowRoute(ctx, typeID, itemID)
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

func (s *Repository) messageRowRoute(ctx context.Context, typeID CommentType, itemID int64) ([]string, error) {
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

		switch itemTypeID { //nolint:exhaustive
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

func (s *Repository) CleanupDeleted(ctx context.Context) (int64, error) {
	query := `
		SELECT cm1.id, cm1.item_id, cm1.type_id
		FROM comment_message AS cm1
			LEFT JOIN comment_message AS cm2 ON cm1.id = cm2.parent_id
		WHERE cm2.parent_id IS NULL
		  AND cm1.delete_date < DATE_SUB(NOW(), INTERVAL ? DAY)
    `

	rows, err := s.db.QueryContext(ctx, query, deleteTTLDays)
	if err != nil {
		return 0, err
	}

	var affected int64

	for rows.Next() {
		var (
			id     int64
			itemID int64
			typeID CommentType
		)

		err = rows.Scan(&id, &itemID, &typeID)
		if err != nil {
			return 0, err
		}

		res, err := s.db.ExecContext(ctx, "DELETE FROM comment_message WHERE id = ?", id)
		if err != nil {
			return 0, err
		}

		a, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}

		affected += a

		err = s.updateTopicStat(ctx, typeID, itemID)
		if err != nil {
			return 0, err
		}
	}

	if err = rows.Err(); err != nil {
		return 0, err
	}

	return affected, nil
}

func (s *Repository) RefreshRepliesCount(ctx context.Context) (int64, error) {
	_, err := s.db.ExecContext(ctx, `
		create temporary table __cms
		select type_id, item_id, parent_id as id, count(1) as count
		from comment_message
		where parent_id is not null
		group by type_id, item_id, parent_id
    `)
	if err != nil {
		return 0, err
	}

	res, err := s.db.ExecContext(ctx, `
		update comment_message
		inner join __cms
		using(type_id, item_id, id)
		set comment_message.replies_count = __cms.count
    `)
	if err != nil {
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return affected, nil
}

func (s *Repository) userURL(userID int64, identity string, language string) (string, error) {
	if len(identity) == 0 {
		identity = "user" + strconv.FormatInt(userID, 10)
	}

	uri, err := s.hostManager.URIByLanguage(language)
	if err != nil {
		return "", err
	}

	uri.Path = "/users/" + url.QueryEscape(identity)

	return uri.String(), nil
}

func (s *Repository) NeedWait(ctx context.Context, userID int64) (bool, error) {
	nextMessageTime, err := s.userRepository.NextMessageTime(ctx, userID)
	if err != nil {
		return false, err
	}

	if nextMessageTime.IsZero() {
		return false, nil
	}

	return time.Now().Before(nextMessageTime), nil
}

func (s *Repository) CleanBrokenMessages(ctx context.Context) (int64, error) {
	var (
		affected int64
		id       int64
	)

	// pictures
	rows, err := s.db.QueryContext(ctx, `
		SELECT comment_message.id
		FROM comment_message
			LEFT JOIN pictures ON comment_message.item_id = pictures.id
		WHERE pictures.id IS NULL AND comment_message.type_id = ?
    `, TypeIDPictures)
	if err != nil {
		return 0, err
	}

	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return 0, err
		}

		a, err := s.deleteMessage(ctx, id)
		if err != nil {
			return 0, err
		}

		affected += a
	}

	if err = rows.Err(); err != nil {
		return 0, err
	}

	// item
	rows, err = s.db.QueryContext(ctx, `
		SELECT comment_message.id
		FROM comment_message
			LEFT JOIN item ON comment_message.item_id = item.id
		WHERE item.id IS NULL AND comment_message.type_id = ?
    `, TypeIDItems)
	if err != nil {
		return 0, err
	}

	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return 0, err
		}

		a, err := s.deleteMessage(ctx, id)
		if err != nil {
			return 0, err
		}

		affected += a
	}

	if err = rows.Err(); err != nil {
		return 0, err
	}

	// votings
	rows, err = s.db.QueryContext(ctx, `
		SELECT comment_message.id
		FROM comment_message
			LEFT JOIN voting ON comment_message.item_id = voting.id
		WHERE voting.id IS NULL AND comment_message.type_id = ?
    `, TypeIDVotings)
	if err != nil {
		return 0, err
	}

	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return 0, err
		}

		a, err := s.deleteMessage(ctx, id)
		if err != nil {
			return 0, err
		}

		affected += a
	}

	if err = rows.Err(); err != nil {
		return 0, err
	}

	// articles
	rows, err = s.db.QueryContext(ctx, `
		SELECT comment_message.id
		FROM comment_message
			LEFT JOIN articles ON comment_message.item_id = articles.id
		WHERE articles.id IS NULL AND comment_message.type_id = ?
    `, TypeIDArticles)
	if err != nil {
		return 0, err
	}

	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return 0, err
		}

		a, err := s.deleteMessage(ctx, id)
		if err != nil {
			return 0, err
		}

		affected += a
	}

	if err = rows.Err(); err != nil {
		return 0, err
	}

	// forums
	rows, err = s.db.QueryContext(ctx, `
		SELECT comment_message.id
		FROM comment_message
			LEFT JOIN forums_topics ON comment_message.item_id = forums_topics.id
		WHERE forums_topics.id IS NULL AND comment_message.type_id = ?
    `, TypeIDForums)
	if err != nil {
		return 0, err
	}

	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return 0, err
		}

		a, err := s.deleteMessage(ctx, id)
		if err != nil {
			return 0, err
		}

		affected += a
	}

	if err = rows.Err(); err != nil {
		return 0, err
	}

	return affected, nil
}

func (s *Repository) deleteMessage(ctx context.Context, id int64) (int64, error) {
	var typeID CommentType

	err := s.db.QueryRowContext(ctx, "SELECT type_id FROM comment_message WHERE id = ?", id).Scan(&typeID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	res, err := s.db.ExecContext(ctx, `
		DELETE FROM comment_message
		WHERE id = ?
    `, id)
	if err != nil {
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	err = s.updateTopicStat(ctx, typeID, id)
	if err != nil {
		return 0, err
	}

	return affected, nil
}

func (s *Repository) CleanTopics(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		DELETE comment_topic_view
		FROM comment_topic_view
			LEFT JOIN comment_message 
				ON comment_topic_view.item_id = comment_message.item_id
				AND comment_topic_view.type_id = comment_message.type_id
		WHERE comment_message.type_id IS NULL
    `)
	if err != nil {
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	res, err = s.db.ExecContext(ctx, `
		DELETE comment_topic
		FROM comment_topic
			LEFT JOIN comment_message 
				ON comment_topic.item_id = comment_message.item_id
				AND comment_topic.type_id = comment_message.type_id
		WHERE comment_message.type_id IS NULL
    `)
	if err != nil {
		return 0, err
	}

	a, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	affected += a

	return affected, nil
}

func (s *Repository) TopicStat(ctx context.Context, typeID CommentType, itemID int64) (int32, error) {
	sqSelect := s.db.Select("messages").From("comment_topic").
		Where(
			goqu.I("comment_topic.type_id").Eq(typeID),
			goqu.I("comment_topic.item_id").Eq(itemID),
		)

	var messages int32

	success, err := sqSelect.ScanValContext(ctx, &messages)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, nil
	}

	return messages, nil
}

func (s *Repository) MessagesCountFromTimestamp(
	ctx context.Context, typeID CommentType, itemID int64, timestamp time.Time,
) (int32, error) {
	sqSelect := s.db.Select(goqu.COUNT(goqu.L("1"))).From(tableCommentMessage).
		Where(
			goqu.I("item_id").Eq(itemID),
			goqu.I("type_id").Eq(typeID),
			goqu.I("datetime").Gt(timestamp),
		)

	var cnt int32

	success, err := sqSelect.ScanValContext(ctx, &cnt)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, nil
	}

	return cnt, nil
}

func (s *Repository) TopicStatForUser(
	ctx context.Context, typeID CommentType, itemID int64, userID int64,
) (int32, int32, error) {
	sqSelect := s.db.Select("comment_topic.messages", "comment_topic_view.timestamp").From("comment_topic").
		LeftJoin(goqu.I("comment_topic_view"), goqu.On(
			goqu.I("comment_topic.type_id").Eq(goqu.I("comment_topic_view.type_id")),
			goqu.I("comment_topic.item_id").Eq(goqu.I("comment_topic_view.item_id")),
			goqu.I("comment_topic_view.user_id").Eq(userID),
		)).
		Where(
			goqu.I("comment_topic.type_id").Eq(typeID),
			goqu.I("comment_topic.item_id").Eq(itemID),
		)

	var messages struct {
		Messages  int32
		Timestamp sql.NullTime
	}

	success, err := sqSelect.ScanStructContext(ctx, &messages)
	if err != nil {
		return 0, 0, err
	}

	if !success {
		return 0, 0, nil
	}

	newMessages := messages.Messages
	if messages.Timestamp.Valid {
		newMessages, err = s.MessagesCountFromTimestamp(ctx, typeID, itemID, messages.Timestamp.Time)
		if err != nil {
			return 0, 0, err
		}
	}

	return messages.Messages, newMessages, nil
}

func (s *Repository) Count(
	ctx context.Context, attention ModeratorAttention, commentType CommentType, itemID int64,
) (int32, error) {
	sqSelect := s.db.Select(goqu.COUNT(goqu.Star())).
		From(tableCommentMessage).
		Where(
			goqu.I(colModeratorAttention).Eq(attention),
			goqu.I("type_id").Eq(commentType),
		)

	if itemID != 0 {
		sqSelect = sqSelect.
			Join(
				goqu.T("pictures"),
				goqu.On(goqu.T(tableCommentMessage).Col("item_id").Eq(goqu.T("pictures").Col("id"))),
			).
			Join(
				goqu.T("picture_item"),
				goqu.On(goqu.T("pictures").Col("id").Eq(goqu.T("picture_item").Col("picture_id"))),
			).
			Join(
				goqu.T("item_parent_cache"),
				goqu.On(goqu.T("picture_item").Col("item_id").Eq(goqu.T("item_parent_cache").Col("item_id"))),
			).
			Where(goqu.T("item_parent_cache").Col("parent_id").Eq(itemID))
	}

	var cnt int32

	success, err := sqSelect.ScanValContext(ctx, &cnt)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, nil
	}

	return cnt, nil
}

func (s *Repository) MessagePage(
	ctx context.Context, messageID int64, perPage int32,
) (int64, CommentType, int32, error) {
	var (
		err     error
		success bool
	)

	row := struct {
		TypeID   CommentType   `db:"type_id"`
		ItemID   int64         `db:"item_id"`
		ParentID sql.NullInt64 `db:"parent_id"`
		Datetime time.Time     `db:"datetime"`
	}{}

	success, err = s.db.Select("type_id", "item_id", "parent_id", "datetime").
		From(tableCommentMessage).
		Where(goqu.I("id").Eq(messageID)).
		ScanStructContext(ctx, &row)
	if err != nil {
		return 0, 0, 0, err
	}

	if !success {
		return 0, 0, 0, errors.New("message not found")
	}

	parentRow := struct {
		ParentID sql.NullInt64 `db:"parent_id"`
		Datetime time.Time     `db:"datetime"`
	}{}
	parentRow.ParentID = row.ParentID
	parentRow.Datetime = row.Datetime

	for success && parentRow.ParentID.Valid {
		success, err = s.db.Select("parent_id", "datetime").From(tableCommentMessage).Where(
			goqu.I("item_id").Eq(row.ItemID),
			goqu.I("type_id").Eq(row.TypeID),
			goqu.I("id").Eq(parentRow.ParentID.Int64),
		).ScanStructContext(ctx, &parentRow)
		if err != nil {
			return 0, 0, 0, err
		}
	}

	var count int32

	success, err = s.db.Select(goqu.COUNT(goqu.L("1"))).From(tableCommentMessage).Where(
		goqu.I("item_id").Eq(row.ItemID),
		goqu.I("type_id").Eq(row.TypeID),
		goqu.I("datetime").Lt(parentRow.Datetime),
		goqu.I("parent_id").IsNull(),
	).ScanValContext(ctx, &count)
	if err != nil || !success {
		return 0, 0, 0, err
	}

	return row.ItemID, row.TypeID, int32(math.Ceil(float64(count+1) / float64(perPage))), nil
}

func (s *Repository) columns(fetchMessage bool, fetchVote bool, fetchIP bool) []interface{} {
	columns := []interface{}{
		goqu.T(tableCommentMessage).Col("id"), goqu.T(tableCommentMessage).Col("type_id"),
		goqu.T(tableCommentMessage).Col("item_id"), goqu.T(tableCommentMessage).Col("parent_id"),
		goqu.T(tableCommentMessage).Col("datetime"), goqu.T(tableCommentMessage).Col("deleted"),
		goqu.T(tableCommentMessage).Col(colModeratorAttention), goqu.T(tableCommentMessage).Col("author_id"),
	}

	if fetchIP {
		columns = append(columns, goqu.T(tableCommentMessage).Col("ip"))
	}

	if fetchMessage {
		columns = append(columns, goqu.T(tableCommentMessage).Col("message"))
	}

	if fetchVote {
		columns = append(columns, goqu.T(tableCommentMessage).Col("vote"))
	}

	return columns
}

func (s *Repository) Message(
	ctx context.Context, messageID int64, fetchMessage bool, fetchVote bool, canViewIP bool,
) (*CommentMessage, error) {
	row := CommentMessage{}

	columns := s.columns(fetchMessage, fetchVote, canViewIP)

	success, err := s.db.Select(columns...).
		From(tableCommentMessage).
		Where(goqu.T(tableCommentMessage).Col("id").Eq(messageID)).
		ScanStructContext(ctx, &row)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, nil //nolint:nilnil
	}

	return &row, nil
}

func (s *Repository) IsNewMessage(
	ctx context.Context, typeID CommentType, itemID int64, msgTime time.Time, userID int64,
) (bool, error) {
	var success bool

	success, err := s.db.Select(goqu.L("1")).
		From("comment_topic_view").
		Where(
			goqu.I("type_id").Eq(typeID),
			goqu.I("item_id").Eq(itemID),
			goqu.I("user_id").Eq(userID),
			goqu.I("timestamp").Gte(msgTime),
		).
		ScanValContext(ctx, &success)
	if err != nil {
		return false, err
	}

	return !success, nil
}

func (s *Repository) MessageRowRoute(
	ctx context.Context, typeID CommentType, itemID int64, messageID int64,
) ([]string, error) {
	var result []string

	switch typeID {
	case TypeIDPictures:
		var identity string

		success, err := s.db.Select("identity").From("pictures").Where(goqu.I("id").Eq(itemID)).
			ScanValContext(ctx, &identity)
		if err != nil {
			return nil, err
		}

		if !success {
			return nil, fmt.Errorf("picture `%v` not found", itemID)
		}

		result = []string{"/picture", identity}

	case TypeIDItems:
		var itemTypeID items.ItemType

		success, err := s.db.Select("item_type_id").From("item").Where(goqu.I("id").Eq(itemID)).
			ScanValContext(ctx, &itemTypeID)
		if err != nil {
			return nil, err
		}

		if !success {
			return nil, fmt.Errorf("item `%v` not found", itemID)
		}

		switch itemTypeID { //nolint:exhaustive
		case items.TWINS:
			result = []string{"/twins", "group", strconv.FormatInt(itemID, 10)}
		case items.MUSEUM:
			result = []string{"/museums", strconv.FormatInt(itemID, 10)}
		default:
			return nil, fmt.Errorf("failed to build url form message `%v` item_type `%v`", itemID, itemTypeID)
		}

	case TypeIDVotings:
		result = []string{"/voting", strconv.FormatInt(itemID, 10)}

	case TypeIDArticles:
		var catname string

		success, err := s.db.Select("catname").From("articles").Where(goqu.I("id").Eq(itemID)).
			ScanValContext(ctx, &catname)
		if err != nil {
			return nil, err
		}

		if !success {
			return nil, fmt.Errorf("article `%v` not found", itemID)
		}

		result = []string{"/articles", catname}

	case TypeIDForums:
		result = []string{"/forums", "message", strconv.FormatInt(messageID, 10)}

	default:
		return nil, fmt.Errorf("unknown type_id `%v`", typeID)
	}

	return result, nil
}

func (s *Repository) Paginator(request Request) *util.Paginator {
	columns := s.columns(request.FetchMessage, request.FetchVote, request.FetchIP)

	sqSelect := s.db.Select(columns...).
		From(tableCommentMessage)

	if request.ItemID > 0 {
		sqSelect = sqSelect.Where(goqu.T(tableCommentMessage).Col("item_id").Eq(request.ItemID))
	}

	if request.TypeID > 0 {
		sqSelect = sqSelect.Where(goqu.T(tableCommentMessage).Col("type_id").Eq(request.TypeID))
	}

	if request.ParentID > 0 {
		sqSelect = sqSelect.Where(goqu.T(tableCommentMessage).Col("parent_id").Eq(request.ParentID))
	}

	if request.PicturesOfItemID > 0 {
		sqSelect = sqSelect.
			Join(goqu.T("pictures"), goqu.On(goqu.T(tableCommentMessage).Col("item_id").Eq(goqu.I("pictures.id")))).
			Join(goqu.T("picture_item"), goqu.On(goqu.I("pictures.id").Eq(goqu.I("picture_item.picture_id")))).
			Join(goqu.T("item_parent_cache"), goqu.On(goqu.I("picture_item.item_id").Eq(goqu.I("item_parent_cache.item_id")))).
			Where(goqu.I("item_parent_cache.parent_id").Eq(request.PicturesOfItemID)).
			Where(goqu.T(tableCommentMessage).Col("type_id").Eq(TypeIDPictures))
	}

	if request.NoParents {
		sqSelect = sqSelect.Where(goqu.T(tableCommentMessage).Col("parent_id").IsNull())
	}

	if request.UserID > 0 {
		sqSelect = sqSelect.Where(goqu.T(tableCommentMessage).Col("author_id").Eq(request.UserID))
	}

	if request.ModeratorAttention > 0 {
		sqSelect = sqSelect.Where(goqu.T(tableCommentMessage).Col(colModeratorAttention).Eq(request.ModeratorAttention))
	}

	sqSelect = sqSelect.Order(request.Order...)

	return &util.Paginator{
		SQLSelect:         sqSelect,
		ItemCountPerPage:  request.PerPage,
		CurrentPageNumber: request.Page,
	}
}
