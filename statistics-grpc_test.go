package goautowp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func assertGridNotEmpty(grid []*PulseGrid) error {
	for _, x := range grid {
		for _, y := range x.Line {
			if y > 0 {
				return nil
			}
		}
	}

	return errors.New("grid is empty")
}

func TestStatisticsPulse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ctxTimeout, cancel := context.WithTimeout(ctx, 5000*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	statisticsClient := NewStatisticsClient(conn)

	cfg := config.LoadConfig(".")
	cnt := NewContainer(cfg)

	db, err := cnt.GoquDB()
	require.NoError(t, err)

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctxTimeout, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	usersClient := NewUsersClient(conn)
	user, err := usersClient.Me(
		metadata.AppendToOutgoingContext(ctxTimeout, authorizationHeader, bearerPrefix+token.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	_, err = db.Insert(schema.TableLogEvents).
		Cols("description", "user_id", "add_datetime").
		Vals(
			goqu.Vals{"Description", user.Id, goqu.Func("NOW")},
		).Executor().Exec()
	require.NoError(t, err)

	r1, err := statisticsClient.GetPulse(ctxTimeout, &PulseRequest{})
	require.NoError(t, err)

	_, err = statisticsClient.GetPulse(ctxTimeout, &PulseRequest{
		Period: PulseRequest_DEFAULT,
	})
	require.NoError(t, err)

	require.NoError(t, assertGridNotEmpty(r1.Grid))

	r1, err = statisticsClient.GetPulse(ctxTimeout, &PulseRequest{
		Period: PulseRequest_MONTH,
	})
	require.NoError(t, err)

	require.NoError(t, assertGridNotEmpty(r1.Grid))

	r1, err = statisticsClient.GetPulse(ctxTimeout, &PulseRequest{
		Period: PulseRequest_YEAR,
	})
	require.NoError(t, err)

	require.NoError(t, assertGridNotEmpty(r1.Grid))
}

func TestAboutData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	statisticsClient := NewStatisticsClient(conn)

	_, err = statisticsClient.GetAboutData(ctx, &emptypb.Empty{})
	require.NoError(t, err)
}

func BenchmarkAboutData(b *testing.B) {
	ctx := context.Background()

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(b, err)

	defer util.Close(conn)
	statisticsClient := NewStatisticsClient(conn)

	for n := 0; n < b.N; n++ {
		_, err = statisticsClient.GetAboutData(ctx, &emptypb.Empty{})
		require.NoError(b, err)
	}
}
