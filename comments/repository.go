package comments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/messaging"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var (
	errUnknownTypeID                          = errors.New("unknown type_id")
	errArticleNotFound                        = errors.New("article not found")
	errMessageNotFound                        = errors.New("message not found")
	errMessageIsDeleted                       = errors.New("message is deleted")
	errAlreadyVoted                           = errors.New("already voted")
	errInvalidType                            = errors.New("invalid type")
	errPictureNotFound                        = errors.New("picture not found")
	errItemNotFound                           = errors.New("item not found")
	errSelfVote                               = errors.New("self-vote forbidden")
	errCommentWithModerAttentionCantBeDeleted = errors.New(
		"comment with moderation attention requirement can't be deleted",
	)
	errFailedToBuildURL = errors.New("failed to build URL")
)

const CommentMessagePreviewLength = 60

const deleteTTLDays = 300

const MaxMessageLength = 16 * 1024

type RatingFan struct {
	UserID int64 `db:"user_id"`
	Volume int64 `db:"volume"`
}

type GetVotesResult struct {
	PositiveVotes []users.DBUser
	NegativeVotes []users.DBUser
}

type Request struct {
	ItemID             int64
	TypeID             schema.CommentMessageType
	ParentID           int64
	NoParents          bool
	UserID             int64
	Order              []exp.OrderedExpression
	ModeratorAttention schema.CommentMessageModeratorAttention
	PicturesOfItemID   int64
	FetchMessage       bool
	FetchVote          bool
	FetchIP            bool
	PerPage            int32
	Page               int32
}

type RatingUser struct {
	AuthorID int64 `db:"author_id"`
	Volume   int64 `db:"volume"`
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
		Executor().QueryContext(ctx) //nolint:sqlclosecheck
	if err != nil {
		return nil, err
	}

	defer util.Close(rows)

	positiveVotes := make([]users.DBUser, 0)
	negativeVotes := make([]users.DBUser, 0)

	for rows.Next() {
		var (
			rUser users.DBUser
			vote  int
		)

		err = rows.Scan(&rUser.ID, &rUser.Name, &rUser.Deleted, &rUser.Identity, &rUser.LastOnline, &rUser.Role,
			&rUser.SpecsWeight, &vote)
		if err != nil {
			return nil, err
		}

		if vote > 0 {
			positiveVotes = append(positiveVotes, rUser)
		} else {
			negativeVotes = append(negativeVotes, rUser)
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
	ctx context.Context, userID int64, commentsType schema.CommentMessageType, itemID int64,
) (bool, error) {
	var result bool

	success, err := s.db.Select(goqu.L("1")).
		From(schema.CommentTopicSubscribeTable).
		Where(
			schema.CommentTopicSubscribeTableTypeIDCol.Eq(commentsType),
			schema.CommentTopicSubscribeTableItemIDCol.Eq(itemID),
			schema.CommentTopicSubscribeTableUserIDCol.Eq(userID),
		).ScanValContext(ctx, &result)

	return success && result, err
}

func (s *Repository) Subscribe(
	ctx context.Context, userID int64, commentsType schema.CommentMessageType, itemID int64,
) error {
	_, err := s.db.Insert(schema.CommentTopicSubscribeTable).Rows(goqu.Record{
		schema.CommentTopicSubscribeTableTypeIDColName: commentsType,
		schema.CommentTopicSubscribeTableItemIDColName: itemID,
		schema.CommentTopicSubscribeTableUserIDColName: userID,
		schema.CommentTopicSubscribeTableSentColName:   false,
	}).OnConflict(goqu.DoNothing()).Executor().ExecContext(ctx)

	return err
}

func (s *Repository) UnSubscribe(
	ctx context.Context, userID int64, commentsType schema.CommentMessageType, itemID int64,
) error {
	_, err := s.db.Delete(schema.CommentTopicSubscribeTable).Where(
		schema.CommentTopicSubscribeTableTypeIDCol.Eq(commentsType),
		schema.CommentTopicSubscribeTableItemIDCol.Eq(itemID),
		schema.CommentTopicSubscribeTableUserIDCol.Eq(userID),
	).Executor().ExecContext(ctx)

	return err
}

func (s *Repository) View(
	ctx context.Context, userID int64, commentsType schema.CommentMessageType, itemID int64,
) error {
	_, err := s.db.Insert(schema.CommentTopicViewTable).Rows(goqu.Record{
		schema.CommentTopicViewTableUserIDColName:    userID,
		schema.CommentTopicViewTableTypeIDColName:    commentsType,
		schema.CommentTopicViewTableItemIDColName:    itemID,
		schema.CommentTopicViewTableTimestampColName: goqu.Func("NOW"),
	}).OnConflict(
		goqu.DoUpdate(
			schema.CommentTopicViewTableUserIDColName+","+
				schema.CommentTopicViewTableTypeIDColName+","+
				schema.CommentTopicViewTableItemIDColName,
			goqu.Record{
				schema.CommentTopicViewTableTimestampColName: goqu.Func("NOW"),
			},
		)).Executor().ExecContext(ctx)

	return err
}

func (s *Repository) QueueDeleteMessage(ctx context.Context, commentID int64, byUserID int64) error {
	var moderatorAttention schema.CommentMessageModeratorAttention

	success, err := s.db.Select(schema.CommentMessageTableModeratorAttentionCol).From(schema.CommentMessageTable).
		Where(schema.CommentMessageTableIDCol.Eq(commentID)).
		ScanValContext(ctx, &moderatorAttention)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	if moderatorAttention == schema.CommentMessageModeratorAttentionRequired {
		return errCommentWithModerAttentionCantBeDeleted
	}

	_, err = s.db.Update(schema.CommentMessageTable).
		Set(goqu.Record{
			schema.CommentMessageTableDeletedColName:    1,
			schema.CommentMessageTableDeletedByColName:  byUserID,
			schema.CommentMessageTableDeleteDateColName: goqu.Func("NOW"),
		}).
		Where(schema.CommentMessageTableIDCol.Eq(commentID)).
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

func (s *Repository) GetCommentType(ctx context.Context, commentID int64) (schema.CommentMessageType, error) {
	var commentType schema.CommentMessageType

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

func (s *Repository) MoveMessage(
	ctx context.Context, commentID int64, dstType schema.CommentMessageType, dstItemID int64,
) error {
	st := struct {
		SrcType   schema.CommentMessageType `db:"type_id"`
		SrcItemID int64                     `db:"item_id"`
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

	_, err = s.db.Update(schema.CommentMessageTable).
		Set(goqu.Record{
			schema.CommentMessageTableTypeIDColName:   dstType,
			schema.CommentMessageTableItemIDColName:   dstItemID,
			schema.CommentMessageTableParentIDColName: nil,
		}).
		Where(schema.CommentMessageTableIDCol.Eq(commentID)).
		Executor().ExecContext(ctx)
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
	dstType schema.CommentMessageType,
	dstItemID int64,
) error {
	_, err := s.db.Update(schema.CommentMessageTable).
		Set(goqu.Record{
			schema.CommentMessageTableTypeIDColName: dstType,
			schema.CommentMessageTableItemIDColName: dstItemID,
		}).
		Where(schema.CommentMessageTableIDCol.Eq(parentID)).
		Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	var ids []int64

	err = s.db.Select(schema.CommentMessageTableIDCol).
		From(schema.CommentMessageTable).
		Where(schema.CommentMessageTableParentIDCol.Eq(parentID)).
		ScanValsContext(ctx, &ids)
	if err != nil {
		return err
	}

	for _, id := range ids {
		err = s.moveMessageRecursive(ctx, id, dstType, dstItemID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) updateTopicStat(ctx context.Context, commentType schema.CommentMessageType, itemID int64) error {
	st := struct {
		MessagesCount int           `db:"count"`
		LastUpdate    *sql.NullTime `db:"last_update"`
	}{}

	success, err := s.db.Select(
		goqu.COUNT(goqu.Star()).As("count"),
		goqu.MAX(schema.CommentMessageTableDatetimeCol).As("last_update"),
	).
		From(schema.CommentMessageTable).
		Where(
			schema.CommentMessageTableTypeIDCol.Eq(commentType),
			schema.CommentMessageTableItemIDCol.Eq(itemID),
		).
		ScanStructContext(ctx, &st)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	if st.MessagesCount <= 0 {
		_, err = s.db.Delete(schema.CommentTopicTable).Where(
			schema.CommentTopicTableTypeIDCol.Eq(commentType),
			schema.CommentTopicTableItemIDCol.Eq(itemID),
		).Executor().ExecContext(ctx)

		return err
	}

	if st.LastUpdate.Valid {
		_, err = s.db.Insert(schema.CommentTopicTable).Rows(goqu.Record{
			schema.CommentTopicTableItemIDColName:     itemID,
			schema.CommentTopicTableTypeIDColName:     commentType,
			schema.CommentTopicTableLastUpdateColName: st.LastUpdate.Time.Format(time.DateTime),
			schema.CommentTopicTableMessagesColName:   st.MessagesCount,
		}).OnConflict(goqu.DoUpdate(
			schema.CommentTopicTableItemIDColName+","+schema.CommentTopicTableTypeIDColName,
			goqu.Record{
				schema.CommentTopicTableLastUpdateColName: goqu.Func("VALUES", goqu.C(schema.CommentTopicTableLastUpdateColName)),
				schema.CommentTopicTableMessagesColName:   goqu.Func("VALUES", goqu.C(schema.CommentTopicTableMessagesColName)),
			},
		)).Executor().ExecContext(ctx)
	}

	return err
}

func (s *Repository) UserVote(ctx context.Context, userID int64, commentID int64) (int32, error) {
	var vote int32

	success, err := s.db.Select(schema.CommentVoteTableVoteCol).
		From(schema.CommentVoteTable).
		Where(schema.CommentVoteTableCommentIDCol.Eq(commentID), schema.CommentVoteTableUserIDCol.Eq(userID)).
		ScanValContext(ctx, &vote)
	if err != nil {
		return 0, err
	}

	if !success {
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
		Where(schema.CommentMessageTableIDCol.Eq(commentID), schema.CommentMessageTableDeletedCol.IsFalse()).
		ScanValContext(ctx, &authorID)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	if authorID == userID {
		return 0, errSelfVote
	}

	res, err := s.db.Insert(schema.CommentVoteTable).Rows(goqu.Record{
		schema.CommentVoteTableCommentIDColName: commentID,
		schema.CommentVoteTableUserIDColName:    userID,
		schema.CommentVoteTableVoteColName:      vote,
	}).OnConflict(goqu.DoUpdate(
		schema.CommentVoteTableCommentIDColName+","+schema.CommentVoteTableUserIDColName,
		goqu.Record{
			schema.CommentVoteTableVoteColName: goqu.Func("VALUES", goqu.C(schema.CommentVoteTableVoteColName)),
		},
	)).Executor().ExecContext(ctx)
	if err != nil {
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	if affected == 0 {
		return 0, errAlreadyVoted
	}

	newVote, err := s.updateVote(ctx, commentID)
	if err != nil {
		return 0, err
	}

	return newVote, nil
}

func (s *Repository) updateVote(ctx context.Context, commentID int64) (int32, error) {
	var count int32

	success, err := s.db.Select(goqu.SUM(schema.CommentVoteTableVoteCol)).
		From(schema.CommentVoteTable).
		Where(schema.CommentVoteTableCommentIDCol.Eq(commentID)).
		Executor().ScanValContext(ctx, &count)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	_, err = s.db.Update(schema.CommentMessageTable).
		Set(goqu.Record{schema.CommentMessageTableVoteColName: count}).
		Where(schema.CommentMessageTableIDCol.Eq(commentID)).
		Executor().ExecContext(ctx)

	return count, err
}

func (s *Repository) CompleteMessage(ctx context.Context, id int64) error {
	_, err := s.db.Update(schema.CommentMessageTable).
		Set(goqu.Record{
			schema.CommentMessageTableModeratorAttentionColName: schema.CommentMessageModeratorAttentionCompleted,
		}).
		Where(
			schema.CommentMessageTableIDCol.Eq(id),
			schema.CommentMessageTableModeratorAttentionCol.Eq(schema.CommentMessageModeratorAttentionRequired),
		).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) Add(
	ctx context.Context,
	typeID schema.CommentMessageType,
	itemID int64,
	parentID int64,
	userID int64,
	message string,
	addr string,
	attention bool,
) (int64, error) {
	if parentID > 0 {
		deleted := false

		success, err := s.db.Select(schema.CommentMessageTableDeletedCol).
			From(schema.CommentMessageTable).
			Where(
				schema.CommentMessageTableTypeIDCol.Eq(typeID),
				schema.CommentMessageTableItemIDCol.Eq(itemID),
				schema.CommentMessageTableIDCol.Eq(parentID),
			).
			ScanValContext(ctx, &deleted)
		if err != nil {
			return 0, err
		}

		if !success {
			return 0, errMessageNotFound
		}

		if deleted {
			return 0, errMessageIsDeleted
		}
	}

	ma := schema.CommentMessageModeratorAttentionNone
	if attention {
		ma = schema.CommentMessageModeratorAttentionRequired
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

	success, err := s.db.Select(goqu.COUNT(goqu.Star())).
		From(schema.CommentMessageTable).
		Where(schema.CommentMessageTableParentIDCol.Eq(messageID)).
		ScanValContext(ctx, &count)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	_, err = s.db.Update(schema.CommentMessageTable).
		Set(goqu.Record{schema.CommentMessageTableRepliesCountColName: count}).
		Where(schema.CommentMessageTableIDCol.Eq(messageID)).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) UpdateTopicView(
	ctx context.Context, typeID schema.CommentMessageType, itemID int64, userID int64,
) error {
	_, err := s.db.Insert(schema.CommentTopicViewTable).Rows(goqu.Record{
		schema.CommentTopicViewTableUserIDColName:    userID,
		schema.CommentTopicViewTableTypeIDColName:    typeID,
		schema.CommentTopicViewTableItemIDColName:    itemID,
		schema.CommentTopicViewTableTimestampColName: goqu.Func("NOW"),
	}).OnConflict(goqu.DoUpdate(
		schema.CommentTopicViewTableUserIDColName+
			","+schema.CommentTopicViewTableTypeIDColName+
			","+schema.CommentTopicViewTableItemIDColName,
		goqu.Record{
			schema.CommentTopicViewTableTimestampColName: goqu.Func("NOW"),
		},
	)).Executor().ExecContext(ctx)

	return err
}

func (s *Repository) AssertItem(ctx context.Context, typeID schema.CommentMessageType, itemID int64) error {
	var (
		err     error
		val     int
		success bool
	)

	switch typeID {
	case schema.CommentMessageTypeIDPictures:
		success, err = s.db.Select(goqu.L("1")).From(schema.PictureTable).
			Where(schema.PictureTableIDCol.Eq(itemID)).ScanValContext(ctx, &val)

	case schema.CommentMessageTypeIDItems:
		success, err = s.db.Select(goqu.L("1")).From(schema.ItemTable).
			Where(schema.ItemTableIDCol.Eq(itemID)).ScanValContext(ctx, &val)

	case schema.CommentMessageTypeIDVotings:
		success, err = s.db.Select(goqu.L("1")).From(schema.VotingTable).
			Where(schema.VotingTableIDCol.Eq(itemID)).ScanValContext(ctx, &val)

	case schema.CommentMessageTypeIDArticles:
		success, err = s.db.Select(goqu.L("1")).From(schema.ArticlesTable).
			Where(schema.ArticlesTableIDCol.Eq(itemID)).ScanValContext(ctx, &val)

	case schema.CommentMessageTypeIDForums:
		success, err = s.db.Select(goqu.L("1")).From(schema.ForumsTopicsTable).
			Where(schema.ForumsTopicsTableIDCol.Eq(itemID)).ScanValContext(ctx, &val)

	default:
		return errInvalidType
	}

	if !success {
		return sql.ErrNoRows
	}

	return err
}

func (s *Repository) NotifyAboutReply(ctx context.Context, messageID int64) error {
	st := struct {
		AuthorID       int64          `db:"author_id"`
		ParentAuthorID int64          `db:"parent_author_id"`
		AuthorIdentity sql.NullString `db:"identity"`
		ParentLanguage string         `db:"language"`
	}{}

	cm1 := "cm1"
	cm1Table := goqu.T(cm1)
	pm := "pm"
	pmTable := goqu.T(pm)
	u1 := "u1"
	u1Table := goqu.T(u1)
	pu := "pu"
	puTable := goqu.T(pu)

	success, err := s.db.Select(
		cm1Table.Col(schema.CommentMessageTableAuthorIDColName),
		pmTable.Col(schema.CommentMessageTableAuthorIDColName).As("parent_author_id"),
		u1Table.Col(schema.UserTableIdentityColName),
		puTable.Col(schema.UserTableLanguageColName),
	).
		From(schema.CommentMessageTable.As(cm1)).
		Join(schema.CommentMessageTable.As(pm), goqu.On(
			cm1Table.Col(schema.CommentMessageTableParentIDColName).Eq(
				pmTable.Col(schema.CommentMessageTableIDColName),
			),
		)).
		Join(schema.UserTable.As(u1), goqu.On(
			cm1Table.Col(schema.CommentMessageTableAuthorIDColName).Eq(u1Table.Col(schema.UserTableIDColName))),
		).
		Join(schema.UserTable.As(pu), goqu.On(
			pmTable.Col(schema.CommentMessageTableAuthorIDColName).Eq(puTable.Col(schema.UserTableIDColName))),
		).
		Where(
			cm1Table.Col(schema.CommentMessageTableIDColName).Eq(messageID),
			cm1Table.Col(schema.CommentMessageTableAuthorIDColName).Neq(pmTable.Col(schema.CommentMessageTableAuthorIDColName)),
			puTable.Col(schema.UserTableDeletedColName).IsFalse(),
		).Executor().ScanStructContext(ctx, &st)
	if err != nil {
		return err
	}

	if !success {
		return nil
	}

	ai := ""
	if st.AuthorIdentity.Valid {
		ai = st.AuthorIdentity.String
	}

	userURL, err := s.userURL(st.AuthorID, ai, st.ParentLanguage)
	if err != nil {
		return err
	}

	uri, err := s.hostManager.URIByLanguage(st.ParentLanguage)
	if err != nil {
		return err
	}

	messageURL, err := s.messageURL(ctx, messageID, uri)
	if err != nil {
		return err
	}

	localizer := s.i18n.Localizer(st.ParentLanguage)

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

	return s.messageRepository.CreateMessage(ctx, 0, st.ParentAuthorID, message)
}

func (s *Repository) NotifySubscribers(ctx context.Context, messageID int64) error {
	var authorIdentity sql.NullString

	st := struct {
		ItemID   int64                     `db:"item_id"`
		TypeID   schema.CommentMessageType `db:"type_id"`
		AuthorID sql.NullInt64             `db:"author_id"`
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
		Executor().QueryContext(ctx) //nolint:sqlclosecheck
	if err != nil {
		return err
	}

	defer util.Close(subscribers)

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
	typeID schema.CommentMessageType,
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
	typeID schema.CommentMessageType,
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
	st := struct {
		ItemID int64                     `db:"item_id"`
		TypeID schema.CommentMessageType `db:"type_id"`
	}{}

	success, err := s.db.Select(schema.CommentMessageTableItemIDCol, schema.CommentMessageTableTypeIDCol).
		From(schema.CommentMessageTable).
		Where(schema.CommentMessageTableIDCol.Eq(messageID)).
		ScanStructContext(ctx, &st)
	if err != nil {
		return "", err
	}

	if !success {
		return "", sql.ErrNoRows
	}

	route, err := s.messageRowRoute(ctx, st.TypeID, st.ItemID)
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

func (s *Repository) messageRowRoute(
	ctx context.Context, typeID schema.CommentMessageType, itemID int64,
) ([]string, error) {
	switch typeID {
	case schema.CommentMessageTypeIDPictures:
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

	case schema.CommentMessageTypeIDItems:
		var itemTypeID schema.ItemTableItemTypeID

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
		case schema.ItemTableItemTypeIDTwins:
			return []string{"/twins", "group", strconv.FormatInt(itemID, 10)}, nil
		case schema.ItemTableItemTypeIDMuseum:
			return []string{"/museums", strconv.FormatInt(itemID, 10)}, nil
		default:
			return nil, fmt.Errorf("%w: for message `%v` item_type `%v`", errFailedToBuildURL, itemID, itemTypeID)
		}

	case schema.CommentMessageTypeIDVotings:
		return []string{"/voting", strconv.FormatInt(itemID, 10)}, nil

	case schema.CommentMessageTypeIDArticles:
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

	case schema.CommentMessageTypeIDForums:
		return []string{"/forums", "message", strconv.FormatInt(itemID, 10)}, nil
	}

	return nil, fmt.Errorf("%w: `%v`", errUnknownTypeID, typeID)
}

func (s *Repository) CleanupDeleted(ctx context.Context) (int64, error) {
	cm1 := schema.CommentMessageTable.As("cm1")
	cm2 := schema.CommentMessageTable.As("cm2")
	cm1ID := cm1.Col(schema.CommentMessageTableIDColName)
	cm2ParentID := cm2.Col(schema.CommentMessageTableParentIDColName)

	rows, err := s.db.Select(
		cm1ID,
		cm1.Col(schema.CommentMessageTableItemIDColName),
		cm1.Col(schema.CommentMessageTableTypeIDColName),
	).
		From(cm1).
		LeftJoin(cm2, goqu.On(cm1ID.Eq(cm2ParentID))).
		Where(
			cm2.Col(schema.CommentMessageTableParentIDColName).IsNull(),
			cm2ParentID.Lt(goqu.L("DATE_SUB(NOW(), INTERVAL ? DAY)", deleteTTLDays)),
		).
		Executor().QueryContext(ctx) //nolint:sqlclosecheck
	if err != nil {
		return 0, err
	}

	defer util.Close(rows)

	var affected int64

	for rows.Next() {
		var (
			id     int64
			itemID int64
			typeID schema.CommentMessageType
		)

		err = rows.Scan(&id, &itemID, &typeID)
		if err != nil {
			return 0, err
		}

		res, err := s.db.Delete(schema.CommentMessageTable).
			Where(schema.CommentMessageTableIDCol.Eq(id)).
			Executor().ExecContext(ctx)
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
			schema.CommentMessageTableTypeIDCol.Eq(schema.CommentMessageTypeIDPictures),
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
			schema.CommentMessageTableTypeIDCol.Eq(schema.CommentMessageTypeIDItems),
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
			schema.CommentMessageTableTypeIDCol.Eq(schema.CommentMessageTypeIDArticles),
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
			schema.CommentMessageTableTypeIDCol.Eq(schema.CommentMessageTypeIDArticles),
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
			goqu.On(schema.CommentMessageTableItemIDCol.Eq(schema.ForumsTopicsTableIDCol)),
		).
		Where(
			schema.ForumsTopicsTableIDCol.IsNull(),
			schema.CommentMessageTableTypeIDCol.Eq(schema.CommentMessageTypeIDForums),
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
	var typeID schema.CommentMessageType

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
	res, err := s.db.Delete(schema.CommentTopicViewTable).Where(
		goqu.L(
			"NOT EXISTS (?)",
			s.db.Select(goqu.V(true)).
				From(schema.CommentMessageTable).
				Where(
					schema.CommentTopicViewTableItemIDCol.Eq(schema.CommentMessageTableItemIDCol),
					schema.CommentTopicViewTableTypeIDCol.Eq(schema.CommentMessageTableTypeIDCol),
				),
		),
	).Executor().ExecContext(ctx)
	if err != nil {
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	res, err = s.db.Delete(schema.CommentTopicTable).Where(
		goqu.L(
			"NOT EXISTS (?)",
			s.db.Select(goqu.V(true)).
				From(schema.CommentMessageTable).
				Where(
					schema.CommentTopicTableItemIDCol.Eq(schema.CommentMessageTableItemIDCol),
					schema.CommentTopicTableTypeIDCol.Eq(schema.CommentMessageTableTypeIDCol),
				),
		),
	).Executor().ExecContext(ctx)
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

func (s *Repository) TopicStat(ctx context.Context, typeID schema.CommentMessageType, itemID int64) (int32, error) {
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
	ctx context.Context, typeID schema.CommentMessageType, itemID int64, timestamp time.Time,
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
	ctx context.Context, typeID schema.CommentMessageType, itemID int64, userID int64,
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
	ctx context.Context, attention schema.CommentMessageModeratorAttention, commentType schema.CommentMessageType,
	itemID int64,
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
				schema.PictureItemTable,
				goqu.On(schema.PictureTableIDCol.Eq(schema.PictureItemTablePictureIDCol)),
			).
			Join(
				schema.ItemParentCacheTable,
				goqu.On(schema.PictureItemTableItemIDCol.Eq(schema.ItemParentCacheTableItemIDCol)),
			).
			Where(schema.ItemParentCacheTableParentIDCol.Eq(itemID))
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
) (int64, schema.CommentMessageType, int32, error) {
	var (
		err     error
		success bool
	)

	row := struct {
		TypeID   schema.CommentMessageType `db:"type_id"`
		ItemID   int64                     `db:"item_id"`
		ParentID sql.NullInt64             `db:"parent_id"`
		Datetime time.Time                 `db:"datetime"`
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
		return 0, 0, 0, errMessageNotFound
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
) (*schema.CommentMessageRow, error) {
	row := schema.CommentMessageRow{}

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
	ctx context.Context, typeID schema.CommentMessageType, itemID int64, msgTime time.Time, userID int64,
) (bool, error) {
	var success bool

	success, err := s.db.Select(goqu.L("1")).
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
	ctx context.Context, typeID schema.CommentMessageType, itemID int64, messageID int64,
) ([]string, error) {
	var result []string

	switch typeID {
	case schema.CommentMessageTypeIDPictures:
		var identity string

		success, err := s.db.Select(schema.PictureTableIdentityCol).From(schema.PictureTable).
			Where(schema.PictureTableIDCol.Eq(itemID)).
			ScanValContext(ctx, &identity)
		if err != nil {
			return nil, err
		}

		if !success {
			return nil, fmt.Errorf("%w: `%v`", errPictureNotFound, itemID)
		}

		result = []string{"/picture", identity}

	case schema.CommentMessageTypeIDItems:
		var itemTypeID schema.ItemTableItemTypeID

		success, err := s.db.Select(schema.ItemTableItemTypeIDCol).
			From(schema.ItemTable).
			Where(schema.ItemTableIDCol.Eq(itemID)).
			ScanValContext(ctx, &itemTypeID)
		if err != nil {
			return nil, err
		}

		if !success {
			return nil, fmt.Errorf("%w: `%v`", errItemNotFound, itemID)
		}

		switch itemTypeID { //nolint:exhaustive
		case schema.ItemTableItemTypeIDTwins:
			result = []string{"/twins", "group", strconv.FormatInt(itemID, 10)}
		case schema.ItemTableItemTypeIDMuseum:
			result = []string{"/museums", strconv.FormatInt(itemID, 10)}
		default:
			return nil, fmt.Errorf("%w: for message `%v` item_type `%v`", errFailedToBuildURL, itemID, itemTypeID)
		}

	case schema.CommentMessageTypeIDVotings:
		result = []string{"/voting", strconv.FormatInt(itemID, 10)}

	case schema.CommentMessageTypeIDArticles:
		var catname string

		success, err := s.db.Select(schema.ArticlesTableCatnameCol).
			From(schema.ArticlesTable).
			Where(schema.ArticlesTableIDCol.Eq(itemID)).
			ScanValContext(ctx, &catname)
		if err != nil {
			return nil, err
		}

		if !success {
			return nil, fmt.Errorf("%w: `%v`", errArticleNotFound, itemID)
		}

		result = []string{"/articles", catname}

	case schema.CommentMessageTypeIDForums:
		result = []string{"/forums", "message", strconv.FormatInt(messageID, 10)}

	default:
		return nil, fmt.Errorf("%w: `%v`", errUnknownTypeID, typeID)
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
		sqSelect = sqSelect.
			Join(
				schema.PictureTable,
				goqu.On(schema.CommentMessageTableItemIDCol.Eq(schema.PictureTableIDCol)),
			).
			Join(
				schema.PictureItemTable,
				goqu.On(schema.PictureTableIDCol.Eq(schema.PictureItemTablePictureIDCol)),
			).
			Join(
				schema.ItemParentCacheTable,
				goqu.On(schema.PictureItemTableItemIDCol.Eq(schema.ItemParentCacheTableItemIDCol)),
			).
			Where(schema.ItemParentCacheTableParentIDCol.Eq(request.PicturesOfItemID)).
			Where(schema.CommentMessageTableTypeIDCol.Eq(schema.CommentMessageTypeIDPictures))
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

func (s *Repository) TopAuthors(ctx context.Context, limit uint) ([]RatingUser, error) {
	rows := make([]RatingUser, 0)

	const volumeAlias = "volume"
	err := s.db.Select(schema.CommentMessageTableAuthorIDCol, goqu.SUM(schema.CommentMessageTableVoteCol).As(volumeAlias)).
		From(schema.CommentMessageTable).
		GroupBy(schema.CommentMessageTableAuthorIDCol).
		Order(goqu.C(volumeAlias).Desc()).
		Limit(limit).
		ScanStructsContext(ctx, &rows)

	return rows, err
}

func (s *Repository) AuthorsFans(ctx context.Context, userID int64, limit uint) ([]RatingFan, error) {
	rows := make([]RatingFan, 0)

	const volumeAlias = "volume"
	err := s.db.Select(schema.CommentVoteTableUserIDCol, goqu.COUNT(goqu.Star()).As(volumeAlias)).
		From(schema.CommentVoteTable).
		Join(schema.CommentMessageTable, goqu.On(schema.CommentVoteTableCommentIDCol.Eq(schema.CommentMessageTableIDCol))).
		Where(schema.CommentMessageTableAuthorIDCol.Eq(userID)).
		GroupBy(schema.CommentVoteTableUserIDCol).
		Order(goqu.C(volumeAlias).Desc()).
		Limit(limit).
		ScanStructsContext(ctx, &rows)

	return rows, err
}
