package goautowp

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
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
	db *sql.DB,
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

	_, err = db.Exec("UPDATE users SET last_message_time = '2000-01-01' WHERE id = ?", user.Id)
	require.NoError(t, err)

	return user.Id, token.AccessToken
}

func TestAddEmptyCommentShouldReturnError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewCommentsClient(conn)
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	_, token := getUserWithCleanHistory(t, conn, cfg, db, testUsername, testPassword)

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
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewCommentsClient(conn)
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	_, token := getUserWithCleanHistory(t, conn, cfg, db, testUsername, testPassword)

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
}

func TestCommentReplyNotificationShouldBeDelivered(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewCommentsClient(conn)
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	_, user1Token := getUserWithCleanHistory(t, conn, cfg, db, testUsername, testPassword)
	_, user2Token := getUserWithCleanHistory(t, conn, cfg, db, adminUsername, adminPassword)

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
