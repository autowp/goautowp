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
	"github.com/autowp/goautowp/schema"
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
	rows, err := s.db.Select(
		schema.UserTableIDCol, schema.UserTableNameCol, schema.UserTableDeletedCol, schema.UserTableIdentityCol,
		schema.UserTableLastOnlineCol, schema.UserTableRoleCol, schema.UserTableSpecsWeightCol,
		schema.CommentVoteTableVoteCol,
	).
		From(schema.CommentVoteTable).
		Join(schema.UserTable, goqu.On(schema.CommentVoteTableUserIDCol.Eq(schema.UserTableIDCol))).
		Where(schema.CommentVoteTableCommentIDCol.Eq(id)).
		Executor().QueryContext(ctx)
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
		SELECT 1 FROM `+schema.CommentTopicSubscribeTableName+` WHERE type_id = ? AND item_id = ? AND user_id = ?
    `, commentsType, itemID, userID)

	return success && result, err
}

func (s *Repository) Subscribe(ctx context.Context, userID int64, commentsType CommentType, itemID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT IGNORE INTO `+schema.CommentTopicSubscribeTableName+` (type_id, item_id, user_id, sent)
		VALUES (?, ?, ?, 0)
    `, commentsType, itemID, userID)

	return err
}

func (s *Repository) UnSubscribe(ctx context.Context, userID int64, commentsType CommentType, itemID int64) error {
	_, err := s.db.ExecContext(
		ctx,
		"DELETE FROM "+schema.CommentTopicSubscribeTableName+" WHERE type_id = ? AND item_id = ? AND user_id = ?",
		commentsType, itemID, userID,
	)

	return err
}

func (s *Repository) View(ctx context.Context, userID int64, commentsType CommentType, itemID int64) error {
	_, err := s.db.ExecContext(
		ctx,
		`
			INSERT INTO `+schema.CommentTopicViewTableName+` (user_id, type_id, item_id, timestamp)
            VALUES (?, ?, ?, NOW())
            ON DUPLICATE KEY UPDATE timestamp = values(timestamp)
        `,
		userID, commentsType, itemID,
	)

	return err
}

func (s *Repository) QueueDeleteMessage(ctx context.Context, commentID int64, byUserID int64) error {
	var moderatorAttention ModeratorAttention

	success, err := s.db.Select(schema.CommentMessageTableModeratorAttentionCol).From(schema.CommentMessageTable).
		Where(schema.CommentMessageTableIDCol.Eq(commentID)).
		ScanValContext(ctx, &moderatorAttention)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	if moderatorAttention == ModeratorAttentionRequired {
		return errors.New("comment with moderation attention requirement can't be deleted")
	}

	_, err = s.db.Update(schema.CommentMessageTable).
		Set(goqu.Record{
			"deleted":     1,
			"deleted_by":  byUserID,
			"delete_date": goqu.Func("NOW"),
		}).
		Where(goqu.C("id").Eq(commentID)).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) RestoreMessage(ctx context.Context, commentID int64) error {
	_, err := s.db.Update(schema.CommentMessageTable).
		Set(goqu.Record{
			schema.CommentMessageTableDeletedColName:    0,
			schema.CommentMessageTableDeleteDateColName: nil,
		}).
		Where(schema.CommentMessageTableIDCol.Eq(commentID)).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) GetCommentType(ctx context.Context, commentID int64) (CommentType, error) {
	var commentType CommentType

	success, err := s.db.Select(schema.CommentMessageTableTypeIDCol).
		From(schema.CommentMessageTable).
		Where(schema.CommentMessageTableIDCol.Eq(commentID)).
		ScanValContext(ctx, &commentType)
	if err != nil {
		return commentType, err
	}

	if !success {
		return commentType, sql.ErrNoRows
	}

	return commentType, nil
}

func (s *Repository) MoveMessage(ctx context.Context, commentID int64, dstType CommentType, dstItemID int64) error {
	st := struct {
		SrcType   CommentType `db:"type_id"`
		SrcItemID int64       `db:"item_id"`
	}{}

	success, err := s.db.Select(schema.CommentMessageTableTypeIDCol, schema.CommentMessageTableItemIDCol).
		From(schema.CommentMessageTable).Where(schema.CommentMessageTableIDCol.Eq(commentID)).
		ScanStructContext(ctx, &st)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	if st.SrcItemID == dstItemID && st.SrcType == dstType {
		return nil
	}

	_, err = s.db.ExecContext(
		ctx,
		"UPDATE "+schema.CommentMessageTableName+" SET type_id = ?, item_id = ?, parent_id = null WHERE id = ?",
		dstType, dstItemID, commentID,
	)
	if err != nil {
		return err
	}

	err = s.moveMessageRecursive(ctx, commentID, dstType, dstItemID)
	if err != nil {
		return err
	}

	err = s.updateTopicStat(ctx, st.SrcType, st.SrcItemID)
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
		"UPDATE "+schema.CommentMessageTableName+" SET type_id = ?, item_id = ? WHERE id = ?",
		dstType, dstItemID, parentID,
	)
	if err != nil {
		return err
	}

	rows, err := s.db.QueryContext(ctx, "SELECT id FROM "+schema.CommentMessageTableName+" WHERE parent_id = ?", parentID)
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
		"SELECT COUNT(1), MAX(datetime) FROM "+schema.CommentMessageTableName+" WHERE type_id = ? AND item_id = ?",
		commentType, itemID,
	).Scan(&messagesCount, &lastUpdate)
	if err != nil {
		return err
	}

	if messagesCount <= 0 {
		_, err = s.db.Delete(schema.CommentTopicTable).Where(
			schema.CommentTopicTableTypeIDCol.Eq(commentType),
			schema.CommentTopicTableItemIDCol.Eq(itemID),
		).Executor().ExecContext(ctx)

		return err
	}

	if lastUpdate.Valid {
		_, err = s.db.ExecContext(
			ctx,
			`
				INSERT INTO `+schema.CommentTopicTableName+` (item_id, type_id, last_update, messages)
				VALUES (?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE last_update = VALUES(last_update), messages = VALUES(messages)
			`,
			itemID, commentType, lastUpdate.Time.Format(time.DateTime), messagesCount,
		)
	}

	return err
}

func (s *Repository) UserVote(ctx context.Context, userID int64, commentID int64) (int32, error) {
	var vote int32
	err := s.db.QueryRowContext(
		ctx, "SELECT vote FROM "+schema.CommentVoteTableName+" WHERE comment_id = ? AND user_id = ?", commentID, userID,
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

	success, err := s.db.Select(schema.CommentMessageTableAuthorIDCol).From(schema.CommentMessageTable).
		Where(schema.CommentMessageTableIDCol.Eq(commentID)).
		ScanValContext(ctx, &authorID)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	if authorID == userID {
		return 0, errors.New("self-vote forbidden")
	}

	res, err := s.db.ExecContext(
		ctx,
		`
            INSERT INTO `+schema.CommentVoteTableName+` (comment_id, user_id, vote)
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
		"SELECT sum(vote) FROM "+schema.CommentVoteTableName+" WHERE comment_id = ?",
		commentID,
	).Scan(&count)
	if err != nil {
		return 0, err
	}

	_, err = s.db.Update(schema.CommentMessageTableName).
		Set(goqu.Record{"vote": count}).
		Where(goqu.C("id").Eq(commentID)).
		Executor().ExecContext(ctx)

	return count, err
}

func (s *Repository) CompleteMessage(ctx context.Context, id int64) error {
	_, err := s.db.Update(schema.CommentMessageTable).
		Set(goqu.Record{schema.CommentMessageTableModeratorAttentionColName: ModeratorAttentionCompleted}).
		Where(
			schema.CommentMessageTableIDCol.Eq(id),
			schema.CommentMessageTableModeratorAttentionCol.Eq(ModeratorAttentionRequired),
		).
		Executor().ExecContext(ctx)

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
			"SELECT deleted FROM "+schema.CommentMessageTableName+" WHERE type_id = ? AND item_id = ? AND id = ?",
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

	res, err := s.db.Insert(schema.CommentMessageTable).
		Cols(schema.CommentMessageTableDatetimeCol, schema.CommentMessageTableTypeIDCol,
			schema.CommentMessageTableItemIDCol, schema.CommentMessageTableParentIDCol,
			schema.CommentMessageTableAuthorIDCol, schema.CommentMessageTableMessageCol,
			schema.CommentMessageTableIPCol, schema.CommentMessageTableModeratorAttentionCol).
		Vals(goqu.Vals{
			goqu.Func("NOW"),
			typeID,
			itemID,
			sql.NullInt64{
				Int64: parentID,
				Valid: parentID > 0,
			},
			userID,
			message,
			goqu.Func("INET6_ATON", addr),
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
		"SELECT count(1) FROM "+schema.CommentMessageTableName+" WHERE parent_id = ?",
		messageID,
	).Scan(&count)
	if err != nil {
		return err
	}

	_, err = s.db.Update(schema.CommentMessageTable).
		Set(goqu.Record{schema.CommentMessageTableRepliesCountColName: count}).
		Where(schema.CommentMessageTableIDCol.Eq(messageID)).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) UpdateTopicView(ctx context.Context, typeID CommentType, itemID int64, userID int64) error {
	_, err := s.db.ExecContext(
		ctx, `
			INSERT INTO `+schema.CommentTopicViewTableName+` (user_id, type_id, item_id, timestamp)
			VALUES (?, ?, ?, NOW())
			ON DUPLICATE KEY UPDATE timestamp = VALUES(timestamp)
		`,
		userID, typeID, itemID,
	)

	return err
}

func (s *Repository) AssertItem(ctx context.Context, typeID CommentType, itemID int64) error {
	var (
		err     error
		val     int
		success bool
	)

	switch typeID {
	case TypeIDPictures:
		success, err = s.db.Select(goqu.L("1")).From(schema.PictureTable).
			Where(schema.PictureTableIDCol.Eq(itemID)).ScanValContext(ctx, &val)

	case TypeIDItems:
		success, err = s.db.Select(goqu.L("1")).From(schema.ItemTable).
			Where(goqu.C("id").Eq(itemID)).ScanValContext(ctx, &val)

	case TypeIDVotings:
		success, err = s.db.Select(goqu.L("1")).From(schema.VotingTable).
			Where(schema.VotingTableIDCol.Eq(itemID)).ScanValContext(ctx, &val)

	case TypeIDArticles:
		success, err = s.db.Select(goqu.L("1")).From(schema.ArticlesTable).
			Where(schema.ArticlesTableIDCol.Eq(itemID)).ScanValContext(ctx, &val)

	case TypeIDForums:
		success, err = s.db.Select(goqu.L("1")).From(schema.ForumsTopicsTable).
			Where(schema.ForumsTopicsTable.Col("id").Eq(itemID)).ScanValContext(ctx, &val)

	default:
		return errors.New("invalid type")
	}

	if !success {
		return sql.ErrNoRows
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
		FROM `+schema.CommentMessageTableName+` AS cm1 
		    JOIN `+schema.CommentMessageTableName+` AS parent_message ON cm1.parent_id = parent_message.id
			JOIN `+schema.UserTableName+` AS u ON cm1.author_id = u.id
			JOIN `+schema.UserTableName+` AS parent_user ON parent_message.author_id = parent_user.id
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
	var authorIdentity sql.NullString

	st := struct {
		ItemID   int64         `db:"item_id"`
		TypeID   CommentType   `db:"type_id"`
		AuthorID sql.NullInt64 `db:"author_id"`
	}{}

	success, err := s.db.Select(schema.CommentMessageTableItemIDCol, schema.CommentMessageTableTypeIDCol,
		schema.CommentMessageTableAuthorIDCol).
		From(schema.CommentMessageTable).Where(schema.CommentMessageTableIDCol.Eq(messageID)).
		ScanStructContext(ctx, &st)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	if !st.AuthorID.Valid {
		return nil
	}

	success, err = s.db.Select(schema.UserTableIdentityCol).
		From(schema.UserTable).
		Where(schema.UserTableIDCol.Eq(st.AuthorID.Int64)).
		ScanValContext(ctx, &authorIdentity)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	au := ""
	if authorIdentity.Valid {
		au = authorIdentity.String
	}

	ids, err := s.getSubscribersIDs(ctx, st.TypeID, st.ItemID, true)
	if err != nil {
		return err
	}

	filteredIDs := make([]int64, 0)

	for _, id := range ids {
		prefs, err := s.userRepository.UserPreferences(ctx, id, st.AuthorID.Int64)
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

	subscribers, err := s.db.Select(schema.UserTableIDCol, schema.UserTableLanguageCol).
		From(schema.UserTable).
		Where(
			schema.UserTableIDCol.In(filteredIDs),
			schema.UserTableIDCol.Neq(st.AuthorID.Int64),
		).
		Executor().QueryContext(ctx)
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

		userURL, err := s.userURL(st.AuthorID.Int64, au, subscriberLanguage)
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

		err = s.SetSubscriptionSent(ctx, st.TypeID, st.ItemID, subscriberID, true)
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
	sel := s.db.Select(schema.CommentTopicSubscribeTableUserIDCol).From(schema.CommentTopicSubscribeTable).Where(
		schema.CommentTopicSubscribeTableTypeIDCol.Eq(typeID),
		schema.CommentTopicSubscribeTableItemIDCol.Eq(itemID),
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
	_, err := s.db.Update(schema.CommentTopicSubscribeTable).
		Set(goqu.Record{"sent": sent}).
		Where(
			schema.CommentTopicSubscribeTableTypeIDCol.Eq(typeID),
			schema.CommentTopicSubscribeTableItemIDCol.Eq(itemID),
			schema.CommentTopicSubscribeTableUserIDCol.Eq(subscriberID),
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
		"SELECT item_id, type_id FROM "+schema.CommentMessageTableName+" WHERE id = ?",
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

		success, err := s.db.Select(schema.PictureTableIdentityCol).
			From(schema.PictureTable).
			Where(schema.PictureTableIDCol.Eq(itemID)).
			ScanValContext(ctx, &identity)
		if err != nil {
			return nil, err
		}

		if !success {
			return nil, sql.ErrNoRows
		}

		return []string{"/picture", identity}, nil

	case TypeIDItems:
		var itemTypeID items.ItemType

		success, err := s.db.Select(schema.ItemTableItemTypeIDCol).
			From(schema.ItemTable).
			Where(schema.ItemTableIDCol.Eq(itemID)).
			ScanValContext(ctx, &itemTypeID)
		if err != nil {
			return nil, err
		}

		if !success {
			return nil, sql.ErrNoRows
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

		success, err := s.db.Select(schema.ArticlesTableCatnameCol).
			From(schema.ArticlesTable).
			Where(schema.ArticlesTableIDCol.Eq(itemID)).
			ScanValContext(ctx, &catname)
		if err != nil {
			return nil, err
		}

		if !success {
			return nil, sql.ErrNoRows
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
		FROM ` + schema.CommentMessageTableName + ` AS cm1
			LEFT JOIN ` + schema.CommentMessageTableName + ` AS cm2 ON cm1.id = cm2.parent_id
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

		res, err := s.db.ExecContext(ctx, "DELETE FROM "+schema.CommentMessageTableName+" WHERE id = ?", id)
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
		from `+schema.CommentMessageTableName+`
		where parent_id is not null
		group by type_id, item_id, parent_id
    `)
	if err != nil {
		return 0, err
	}

	res, err := s.db.ExecContext(ctx, `
		update `+schema.CommentMessageTableName+`
		inner join __cms
		using(type_id, item_id, id)
		set `+schema.CommentMessageTableName+`.replies_count = __cms.count
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
		ids      []int64
	)

	// pictures
	err := s.db.Select(schema.CommentMessageTableIDCol).
		From(schema.CommentMessageTable).
		LeftJoin(
			schema.PictureTable,
			goqu.On(schema.CommentMessageTableItemIDCol.Eq(schema.PictureTableIDCol)),
		).
		Where(
			schema.PictureTableIDCol.IsNull(),
			schema.CommentMessageTableTypeIDCol.Eq(TypeIDPictures),
		).ScanValsContext(ctx, &ids)
	if err != nil {
		return 0, err
	}

	for _, id := range ids {
		a, err := s.deleteMessage(ctx, id)
		if err != nil {
			return 0, err
		}

		affected += a
	}

	// item
	err = s.db.Select(schema.CommentMessageTableIDCol).
		From(schema.CommentMessageTable).
		LeftJoin(
			schema.ItemTable,
			goqu.On(schema.CommentMessageTableItemIDCol.Eq(schema.ItemTableIDCol)),
		).
		Where(
			schema.ItemTableIDCol.IsNull(),
			schema.CommentMessageTableTypeIDCol.Eq(TypeIDItems),
		).ScanValsContext(ctx, &ids)
	if err != nil {
		return 0, err
	}

	for _, id := range ids {
		a, err := s.deleteMessage(ctx, id)
		if err != nil {
			return 0, err
		}

		affected += a
	}

	// votings
	err = s.db.Select(schema.CommentMessageTableIDCol).
		From(schema.CommentMessageTable).
		LeftJoin(
			schema.VotingTable,
			goqu.On(schema.CommentMessageTableItemIDCol.Eq(schema.VotingTableIDCol)),
		).
		Where(
			schema.VotingTableIDCol.IsNull(),
			schema.CommentMessageTableTypeIDCol.Eq(TypeIDArticles),
		).ScanValsContext(ctx, &ids)
	if err != nil {
		return 0, err
	}

	for _, id := range ids {
		a, err := s.deleteMessage(ctx, id)
		if err != nil {
			return 0, err
		}

		affected += a
	}

	// articles
	err = s.db.Select(schema.CommentMessageTableIDCol).
		From(schema.CommentMessageTable).
		LeftJoin(schema.ArticlesTable, goqu.On(schema.CommentMessageTableItemIDCol.Eq(schema.ArticlesTableIDCol))).
		Where(
			schema.ArticlesTableIDCol.IsNull(),
			schema.CommentMessageTableTypeIDCol.Eq(TypeIDArticles),
		).ScanValsContext(ctx, &ids)
	if err != nil {
		return 0, err
	}

	for _, id := range ids {
		a, err := s.deleteMessage(ctx, id)
		if err != nil {
			return 0, err
		}

		affected += a
	}

	// forums
	err = s.db.Select(schema.CommentMessageTableIDCol).
		From(schema.CommentMessageTable).
		LeftJoin(
			schema.ForumsTopicsTable,
			goqu.On(schema.CommentMessageTableItemIDCol.Eq(schema.ForumsTopicsTable.Col("id"))),
		).
		Where(
			schema.ForumsTopicsTable.Col("id").IsNull(),
			schema.CommentMessageTableTypeIDCol.Eq(TypeIDForums),
		).ScanValsContext(ctx, &ids)
	if err != nil {
		return 0, err
	}

	for _, id := range ids {
		a, err := s.deleteMessage(ctx, id)
		if err != nil {
			return 0, err
		}

		affected += a
	}

	return affected, nil
}

func (s *Repository) deleteMessage(ctx context.Context, id int64) (int64, error) {
	var typeID CommentType

	success, err := s.db.Select("type_id").
		From(schema.CommentMessageTable).
		Where(schema.CommentMessageTableIDCol.Eq(id)).
		ScanValContext(ctx, &typeID)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, nil
	}

	res, err := s.db.Delete(schema.CommentMessageTable).
		Where(schema.CommentMessageTableIDCol.Eq(id)).
		Executor().ExecContext(ctx)
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
		DELETE `+schema.CommentTopicViewTableName+`
		FROM `+schema.CommentTopicViewTableName+`
			LEFT JOIN `+schema.CommentMessageTableName+` 
				ON `+schema.CommentTopicViewTableName+`.item_id = `+schema.CommentMessageTableName+`.item_id
				AND `+schema.CommentTopicViewTableName+`.type_id = `+schema.CommentMessageTableName+`.type_id
		WHERE `+schema.CommentMessageTableName+`.type_id IS NULL
    `)
	if err != nil {
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	res, err = s.db.ExecContext(ctx, `
		DELETE `+schema.CommentTopicTableName+`
		FROM `+schema.CommentTopicTableName+`
			LEFT JOIN `+schema.CommentMessageTableName+` 
				ON `+schema.CommentTopicTableName+`.item_id = `+schema.CommentMessageTableName+`.item_id
				AND `+schema.CommentTopicTableName+`.type_id = `+schema.CommentMessageTableName+`.type_id
		WHERE `+schema.CommentMessageTableName+`.type_id IS NULL
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
	sqSelect := s.db.Select(schema.CommentTopicTableMessagesCol).
		From(schema.CommentTopicTable).
		Where(
			schema.CommentTopicTableTypeIDCol.Eq(typeID),
			schema.CommentTopicTableItemIDCol.Eq(itemID),
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
	sqSelect := s.db.Select(goqu.COUNT(goqu.Star())).From(schema.CommentMessageTable).
		Where(
			schema.CommentMessageTableItemIDCol.Eq(itemID),
			schema.CommentMessageTableTypeIDCol.Eq(typeID),
			schema.CommentMessageTableDatetimeCol.Gt(timestamp),
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
	sqSelect := s.db.Select(schema.CommentTopicTableMessagesCol, schema.CommentTopicViewTableTimestampCol).
		From(schema.CommentTopicTable).
		LeftJoin(schema.CommentTopicViewTable, goqu.On(
			schema.CommentTopicTableTypeIDCol.Eq(schema.CommentTopicViewTableTypeIDCol),
			schema.CommentTopicTableItemIDCol.Eq(schema.CommentTopicViewTableItemIDCol),
			schema.CommentTopicViewTableUserIDCol.Eq(userID),
		)).
		Where(
			schema.CommentTopicTableTypeIDCol.Eq(typeID),
			schema.CommentTopicTableItemIDCol.Eq(itemID),
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
		From(schema.CommentMessageTable).
		Where(
			schema.CommentMessageTableModeratorAttentionCol.Eq(attention),
			schema.CommentMessageTableTypeIDCol.Eq(commentType),
		)

	if itemID != 0 {
		sqSelect = sqSelect.
			Join(
				schema.PictureTable,
				goqu.On(schema.CommentMessageTableItemIDCol.Eq(schema.PictureTableIDCol)),
			).
			Join(
				goqu.T(schema.TablePictureItem),
				goqu.On(schema.PictureTableIDCol.Eq(goqu.T(schema.TablePictureItem).Col("picture_id"))),
			).
			Join(
				goqu.T(schema.TableItemParentCache),
				goqu.On(goqu.T(schema.TablePictureItem).Col("item_id").Eq(goqu.T(schema.TableItemParentCache).Col("item_id"))),
			).
			Where(goqu.T(schema.TableItemParentCache).Col("parent_id").Eq(itemID))
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

	success, err = s.db.Select(
		schema.CommentMessageTableTypeIDCol, schema.CommentMessageTableItemIDCol, schema.CommentMessageTableParentIDCol,
		schema.CommentMessageTableDatetimeCol).
		From(schema.CommentMessageTable).
		Where(schema.CommentMessageTableIDCol.Eq(messageID)).
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
		success, err = s.db.Select(schema.CommentMessageTableParentIDCol, schema.CommentMessageTableDatetimeCol).
			From(schema.CommentMessageTable).
			Where(
				schema.CommentMessageTableItemIDCol.Eq(row.ItemID),
				schema.CommentMessageTableTypeIDCol.Eq(row.TypeID),
				schema.CommentMessageTableIDCol.Eq(parentRow.ParentID.Int64),
			).
			ScanStructContext(ctx, &parentRow)
		if err != nil {
			return 0, 0, 0, err
		}
	}

	var count int32

	success, err = s.db.Select(goqu.COUNT(goqu.Star())).From(schema.CommentMessageTable).Where(
		schema.CommentMessageTableItemIDCol.Eq(row.ItemID),
		schema.CommentMessageTableTypeIDCol.Eq(row.TypeID),
		schema.CommentMessageTableDatetimeCol.Lt(parentRow.Datetime),
		schema.CommentMessageTableParentIDCol.IsNull(),
	).ScanValContext(ctx, &count)
	if err != nil || !success {
		return 0, 0, 0, err
	}

	return row.ItemID, row.TypeID, int32(math.Ceil(float64(count+1) / float64(perPage))), nil
}

func (s *Repository) columns(fetchMessage bool, fetchVote bool, fetchIP bool) []interface{} {
	columns := []interface{}{
		schema.CommentMessageTableIDCol, schema.CommentMessageTableTypeIDCol,
		schema.CommentMessageTableItemIDCol, schema.CommentMessageTableParentIDCol,
		schema.CommentMessageTableDatetimeCol, schema.CommentMessageTableDeletedCol,
		schema.CommentMessageTableModeratorAttentionCol, schema.CommentMessageTableAuthorIDCol,
	}

	if fetchIP {
		columns = append(columns, schema.CommentMessageTableIPCol)
	}

	if fetchMessage {
		columns = append(columns, schema.CommentMessageTableMessageCol)
	}

	if fetchVote {
		columns = append(columns, schema.CommentMessageTableVoteCol)
	}

	return columns
}

func (s *Repository) Message(
	ctx context.Context, messageID int64, fetchMessage bool, fetchVote bool, canViewIP bool,
) (*CommentMessage, error) {
	row := CommentMessage{}

	columns := s.columns(fetchMessage, fetchVote, canViewIP)

	success, err := s.db.Select(columns...).
		From(schema.CommentMessageTable).
		Where(schema.CommentMessageTableIDCol.Eq(messageID)).
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

	success, err := s.db.Select(goqu.Star()).
		From(schema.CommentTopicViewTable).
		Where(
			schema.CommentTopicViewTableTypeIDCol.Eq(typeID),
			schema.CommentTopicViewTableItemIDCol.Eq(itemID),
			schema.CommentTopicViewTableUserIDCol.Eq(userID),
			schema.CommentTopicViewTableTimestampCol.Gte(msgTime),
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

		success, err := s.db.Select(schema.PictureTableIdentityCol).From(schema.PictureTable).
			Where(schema.PictureTableIDCol.Eq(itemID)).
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

		success, err := s.db.Select(schema.ItemTableItemTypeIDCol).
			From(schema.ItemTable).
			Where(schema.ItemTableIDCol.Eq(itemID)).
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

		success, err := s.db.Select(schema.ArticlesTableCatnameCol).
			From(schema.ArticlesTable).
			Where(schema.ArticlesTableIDCol.Eq(itemID)).
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
		From(schema.CommentMessageTable)

	if request.ItemID > 0 {
		sqSelect = sqSelect.Where(schema.CommentMessageTableItemIDCol.Eq(request.ItemID))
	}

	if request.TypeID > 0 {
		sqSelect = sqSelect.Where(schema.CommentMessageTableTypeIDCol.Eq(request.TypeID))
	}

	if request.ParentID > 0 {
		sqSelect = sqSelect.Where(schema.CommentMessageTableParentIDCol.Eq(request.ParentID))
	}

	if request.PicturesOfItemID > 0 {
		tablePictureItems := goqu.T(schema.TablePictureItem)
		sqSelect = sqSelect.
			Join(
				schema.PictureTable,
				goqu.On(schema.CommentMessageTableItemIDCol.Eq(schema.PictureTableIDCol)),
			).
			Join(
				tablePictureItems,
				goqu.On(schema.PictureTableIDCol.Eq(tablePictureItems.Col("picture_id"))),
			).
			Join(
				goqu.T(schema.TableItemParentCache),
				goqu.On(tablePictureItems.Col("item_id").Eq(goqu.T(schema.TableItemParentCache).Col("item_id"))),
			).
			Where(goqu.T(schema.TableItemParentCache).Col("parent_id").Eq(request.PicturesOfItemID)).
			Where(schema.CommentMessageTableTypeIDCol.Eq(TypeIDPictures))
	}

	if request.NoParents {
		sqSelect = sqSelect.Where(schema.CommentMessageTableParentIDCol.IsNull())
	}

	if request.UserID > 0 {
		sqSelect = sqSelect.Where(schema.CommentMessageTableAuthorIDCol.Eq(request.UserID))
	}

	if request.ModeratorAttention > 0 {
		sqSelect = sqSelect.Where(schema.CommentMessageTableModeratorAttentionCol.Eq(request.ModeratorAttention))
	}

	sqSelect = sqSelect.Order(request.Order...)

	return &util.Paginator{
		SQLSelect:         sqSelect,
		ItemCountPerPage:  request.PerPage,
		CurrentPageNumber: request.Page,
	}
}
