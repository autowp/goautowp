package goautowp

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

//nolint:unparam
func getUserWithCleanHistory(
	t *testing.T,
	conn *grpc.ClientConn,
	cfg config.Config,
	db *goqu.Database,
	username string,
	password string,
) (int64, string) {
	t.Helper()

	ctx := context.Background()
	kc := gocloak.NewClient(cfg.Keycloak.URL)

	token, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, username, password)
	require.NoError(t, err)
	require.NotNil(t, token)

	user, err := NewUsersClient(conn).Me(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	_, err = db.Update(schema.UserTable).
		Set(goqu.Record{
			"last_message_time": "2000-01-01",
			"votes_left":        100,
		}).
		Where(goqu.C("id").Eq(user.Id)).
		Executor().ExecContext(ctx)
	require.NoError(t, err)

	return user.Id, token.AccessToken
}

func TestAddEmptyCommentShouldReturnError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewCommentsClient(conn)
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	_, token := getUserWithCleanHistory(t, conn, cfg, goquDB, testUsername, testPassword)

	_, err = client.Add(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token),
		&AddCommentRequest{
			ItemId:             1,
			TypeId:             CommentsType_ITEM_TYPE_ID,
			Message:            "",
			ModeratorAttention: true,
			ParentId:           0,
			Resolve:            false,
		},
	)
	require.NotNil(t, err)
}

func TestAddComment(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewCommentsClient(conn)
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	_, token := getUserWithCleanHistory(t, conn, cfg, goquDB, testUsername, testPassword)

	r, err := client.Add(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token),
		&AddCommentRequest{
			ItemId:             1,
			TypeId:             CommentsType_ARTICLES_TYPE_ID,
			Message:            "Test",
			ModeratorAttention: false,
			ParentId:           0,
			Resolve:            false,
		},
	)
	require.NoError(t, err)

	r2, err := client.GetMessage(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token),
		&GetMessageRequest{
			Id: r.Id,
			Fields: &CommentMessageFields{
				Preview:  true,
				Route:    true,
				Text:     true,
				Vote:     true,
				UserVote: true,
				Replies:  true,
				Status:   true,
			},
		},
	)
	require.NoError(t, err)
	require.Equal(t, "Test", r2.Text)

	r3, err := client.GetMessages(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token),
		&GetMessagesRequest{
			ItemId:    r2.ItemId,
			TypeId:    r2.TypeId,
			ParentId:  0,
			NoParents: true,
			UserId:    r2.AuthorId,
			Order:     1,
			Limit:     1,
			Page:      1,
			Fields: &CommentMessageFields{
				Preview:  true,
				Route:    true,
				Text:     true,
				Vote:     true,
				UserVote: true,
				Replies:  true,
				Status:   true,
			},
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, r3.Items)

	_, err = client.View(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token),
		&CommentsViewRequest{
			ItemId: 1,
			TypeId: CommentsType_ARTICLES_TYPE_ID,
		},
	)
	require.NoError(t, err)
}

func TestCommentReplyNotificationShouldBeDelivered(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewCommentsClient(conn)
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	_, user1Token := getUserWithCleanHistory(t, conn, cfg, goquDB, testUsername, testPassword)
	_, user2Token := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	response, err := client.Add(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+user1Token),
		&AddCommentRequest{
			ItemId:             1,
			TypeId:             CommentsType_ARTICLES_TYPE_ID,
			Message:            "Root comment",
			ModeratorAttention: false,
			ParentId:           0,
			Resolve:            false,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, response.Id)

	response, err = client.Add(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+user2Token),
		&AddCommentRequest{
			ItemId:             1,
			TypeId:             CommentsType_ARTICLES_TYPE_ID,
			Message:            "Reply comment",
			ModeratorAttention: false,
			ParentId:           response.Id,
			Resolve:            false,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, response.Id)

	messagesClient := NewMessagingClient(conn)
	messages, err := messagesClient.GetMessages(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+user1Token),
		&MessagingGetMessagesRequest{
			Folder: "system",
			Page:   1,
		},
	)
	require.NoError(t, err)
	require.Contains(t, messages.Items[0].Text, "replies to you")
}

func TestSubscribeComment(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewCommentsClient(conn)
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	_, userToken := getUserWithCleanHistory(t, conn, cfg, goquDB, testUsername, testPassword)

	_, err = client.Subscribe(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+userToken),
		&CommentsSubscribeRequest{
			ItemId: 1,
			TypeId: CommentsType_ARTICLES_TYPE_ID,
		},
	)
	require.NoError(t, err)

	_, err = client.UnSubscribe(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+userToken),
		&CommentsUnSubscribeRequest{
			ItemId: 1,
			TypeId: CommentsType_ARTICLES_TYPE_ID,
		},
	)
	require.NoError(t, err)
}

func TestVoteComment(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewCommentsClient(conn)
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	_, userToken := getUserWithCleanHistory(t, conn, cfg, goquDB, testUsername, testPassword)
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	r, err := client.Add(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+userToken),
		&AddCommentRequest{
			ItemId:             1,
			TypeId:             CommentsType_ARTICLES_TYPE_ID,
			Message:            "Test",
			ModeratorAttention: false,
			ParentId:           0,
			Resolve:            false,
		},
	)
	require.NoError(t, err)

	// vote comment
	_, err = client.VoteComment(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+userToken),
		&CommentsVoteCommentRequest{
			CommentId: r.Id,
			Vote:      1,
		},
	)
	require.ErrorContains(t, err, "self-vote forbidden")

	_, err = client.VoteComment(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&CommentsVoteCommentRequest{
			CommentId: r.Id,
			Vote:      1,
		},
	)
	require.NoError(t, err)

	// get comment votes
	_, err = client.GetCommentVotes(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+userToken),
		&GetCommentVotesRequest{
			CommentId: r.Id,
		},
	)
	require.NoError(t, err)

	// delete comment
	_, err = client.SetDeleted(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+userToken),
		&CommentsSetDeletedRequest{
			CommentId: r.Id,
			Deleted:   true,
		},
	)
	require.ErrorContains(t, err, "PermissionDenied")

	_, err = client.SetDeleted(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&CommentsSetDeletedRequest{
			CommentId: r.Id,
			Deleted:   true,
		},
	)
	require.NoError(t, err)

	// restore comment
	_, err = client.SetDeleted(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+userToken),
		&CommentsSetDeletedRequest{
			CommentId: r.Id,
			Deleted:   false,
		},
	)
	require.ErrorContains(t, err, "PermissionDenied")

	_, err = client.SetDeleted(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&CommentsSetDeletedRequest{
			CommentId: r.Id,
			Deleted:   false,
		},
	)
	require.NoError(t, err)

	// move message
	_, err = client.MoveComment(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&CommentsMoveCommentRequest{
			CommentId: r.Id,
			ItemId:    2,
			TypeId:    CommentsType_ARTICLES_TYPE_ID,
		},
	)
	require.ErrorContains(t, err, "PermissionDenied")
}

func TestCompleteComment(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewCommentsClient(conn)
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	_, userToken := getUserWithCleanHistory(t, conn, cfg, goquDB, testUsername, testPassword)
	_, adminToken := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	r, err := client.Add(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+userToken),
		&AddCommentRequest{
			ItemId:             1,
			TypeId:             CommentsType_ARTICLES_TYPE_ID,
			Message:            "Test",
			ModeratorAttention: true,
			ParentId:           0,
			Resolve:            false,
		},
	)
	require.NoError(t, err)

	_, err = client.Add(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken),
		&AddCommentRequest{
			ItemId:             1,
			TypeId:             CommentsType_ARTICLES_TYPE_ID,
			Message:            "Test",
			ModeratorAttention: true,
			Resolve:            true,
			ParentId:           r.Id,
		},
	)
	require.NoError(t, err)
}

func TestMessagesByUserIdentity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewCommentsClient(conn)
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	_, token := getUserWithCleanHistory(t, conn, cfg, goquDB, adminUsername, adminPassword)

	_, err = client.GetMessages(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token),
		&GetMessagesRequest{
			ItemId:       1,
			TypeId:       CommentsType_ITEM_TYPE_ID,
			UserIdentity: "test",
			Page:         2,
		},
	)
	require.NoError(t, err)
}
