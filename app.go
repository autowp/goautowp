package goautowp

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/items"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"    // enable mysql dialect
	_ "github.com/doug-martin/goqu/v9/dialect/postgres" // enable postgres dialect
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql" // enable mysql driver
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"    // enable mysql migrations
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // enable postgres migrations
	_ "github.com/golang-migrate/migrate/v4/source/file"       // enable file migration source
	_ "github.com/lib/pq"                                      // enable postgres driver
	"github.com/sirupsen/logrus"
)

// Application is Service Main Object.
type Application struct {
	container *Container
}

// NewApplication constructor.
func NewApplication(cfg config.Config) *Application {
	app := &Application{
		container: NewContainer(cfg),
	}

	gin.SetMode(cfg.GinMode)

	return app
}

func (s *Application) MigrateAutowp(_ context.Context) error {
	_, err := s.container.AutowpDB()
	if err != nil {
		return err
	}

	cfg := s.container.Config()

	err = applyMigrations(cfg.AutowpMigrations)
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

func (s *Application) ServeGRPC(quit chan bool) error {
	grpcServer, err := s.container.GRPCServerWithServices()
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", s.container.Config().GRPC.Listen)
	if err != nil {
		return err
	}

	go func() {
		<-quit

		grpcServer.GracefulStop()
	}()

	logrus.Println("gRPC listener started")

	err = grpcServer.Serve(lis)
	if err != nil {
		// cannot panic, because this probably is an intentional close
		logrus.Printf("gRPC: Serve() error: %s", err)
	}

	logrus.Println("gRPC listener stopped")

	return nil
}

func (s *Application) ServePublic(ctx context.Context, quit chan bool) error {
	httpServer, err := s.container.PublicHTTPServer(ctx)
	if err != nil {
		return err
	}

	go func(ctx context.Context) {
		<-quit

		if err := httpServer.Shutdown(ctx); err != nil {
			logrus.Error(err.Error())
		}
	}(ctx)

	logrus.Println("public HTTP listener started")

	err = httpServer.ListenAndServe()
	if err != nil {
		// cannot panic, because this probably is an intentional close
		logrus.Printf("Httpserver: ListenAndServe() error: %s", err)
	}

	logrus.Println("public HTTP listener stopped")

	return nil
}

func (s *Application) ListenDuplicateFinderAMQP(ctx context.Context, quit chan bool) error {
	df, err := s.container.DuplicateFinder()
	if err != nil {
		return err
	}

	cfg := s.container.Config()

	logrus.Println("DuplicateFinder listener started")

	err = df.ListenAMQP(ctx, cfg.DuplicateFinder.RabbitMQ, cfg.DuplicateFinder.Queue, quit)
	if err != nil {
		logrus.Error(err.Error())

		return err
	}

	logrus.Println("DuplicateFinder listener stopped")

	return nil
}

// Close Destructor.
func (s *Application) Close() error {
	logrus.Println("Closing service")

	if err := s.container.Close(); err != nil {
		return err
	}

	logrus.Println("Service closed")

	return nil
}

func applyMigrations(config config.MigrationsConfig) error {
	logrus.Info("Apply migrations")

	dir := config.Dir
	if dir == "" {
		ex, err := os.Executable()
		if err != nil {
			return err
		}

		exPath := filepath.Dir(ex)
		dir = exPath + "/migrations"
	}

	m, err := migrate.New("file://"+dir, config.DSN)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil {
		return err
	}

	logrus.Info("Migrations applied")

	return nil
}

func (s *Application) MigratePostgres(_ context.Context) error {
	_, err := s.container.GoquPostgresDB()
	if err != nil {
		return err
	}

	cfg := s.container.Config()

	err = applyMigrations(cfg.PostgresMigrations)
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

func (s *Application) ServePrivate(ctx context.Context, quit chan bool) error {
	httpServer, err := s.container.PrivateHTTPServer(ctx)
	if err != nil {
		return err
	}

	go func(ctx context.Context) {
		<-quit

		if err := httpServer.Shutdown(ctx); err != nil {
			logrus.Error(err.Error())
		}
	}(ctx)

	logrus.Info("HTTP server started")

	if err = httpServer.ListenAndServe(); err != nil {
		// cannot panic, because this probably is an intentional close
		logrus.Infof("Httpserver: ListenAndServe() error: %s", err)
	}

	logrus.Info("HTTP server stopped")

	return nil
}

func (s *Application) SchedulerHourly(ctx context.Context) error {
	traffic, err := s.container.Traffic()
	if err != nil {
		return err
	}

	deleted, err := traffic.Monitoring.GC(ctx)
	if err != nil {
		logrus.Error(err.Error())

		return err
	}

	logrus.Infof("`%v` items of monitoring deleted", deleted)

	deleted, err = traffic.Ban.GC(ctx)
	if err != nil {
		logrus.Error(err.Error())

		return err
	}

	logrus.Infof("`%v` items of ban deleted", deleted)

	err = traffic.AutoWhitelist(ctx)
	if err != nil {
		logrus.Error(err.Error())

		return err
	}

	return nil
}

func (s *Application) SchedulerDaily(ctx context.Context) error {
	usersRep, err := s.container.UsersRepository()
	if err != nil {
		return err
	}

	err = usersRep.UpdateSpecsVolumes(ctx)
	if err != nil {
		logrus.Error(err.Error())

		return err
	}

	commentsRep, err := s.container.CommentsRepository()
	if err != nil {
		return err
	}

	affected, err := commentsRep.CleanupDeleted(ctx)
	if err != nil {
		logrus.Error(err.Error())

		return err
	}

	logrus.Infof("Comments deleted: %d", affected)

	// affected, err = commentsRep.RefreshRepliesCount(ctx)
	// if err != nil {
	//	logrus.Error(err.Error())
	//
	//	return err
	// }
	//
	// logrus.Infof("Replies refreshed: %d", affected)

	// affected, err = commentsRep.CleanBrokenMessages(ctx)
	// if err != nil {
	//	logrus.Error(err.Error())
	//
	//	return err
	// }
	//
	// logrus.Infof("Clean broken: %d", affected)

	// affected, err = commentsRep.CleanTopics(ctx)
	// if err != nil {
	//	logrus.Error(err.Error())
	//
	//	return err
	// }
	//
	// logrus.Infof("Clean topics: %d", affected)

	return nil
}

func (s *Application) SchedulerMidnight(ctx context.Context) error {
	ur, err := s.container.UsersRepository()
	if err != nil {
		return err
	}

	err = ur.RestoreVotes(ctx)
	if err != nil {
		logrus.Error(err.Error())

		return err
	}

	affected, err := ur.UpdateVotesLimits(ctx)
	if err != nil {
		logrus.Error(err.Error())

		return err
	}

	logrus.Infof("Updated %d users vote limits", affected)

	idr, err := s.container.ItemOfDayRepository()
	if err != nil {
		return err
	}

	success, err := idr.Pick(ctx)
	if err != nil {
		logrus.Error(err.Error())

		return err
	}

	logrus.Infof("item of day status: `%v`", success)

	return nil
}

func (s *Application) Autoban(ctx context.Context, quit chan bool) error {
	traffic, err := s.container.Traffic()
	if err != nil {
		return err
	}

	banTicker := time.NewTicker(time.Minute)

	logrus.Info("AutoBan scheduler started")
loop:
	for {
		select {
		case <-banTicker.C:
			err := traffic.AutoBan(ctx)
			if err != nil {
				logrus.Error(err.Error())
			}
		case <-quit:
			banTicker.Stop()

			break loop
		}
	}

	logrus.Info("AutoBan scheduler stopped")

	return nil
}

func (s *Application) generateBrandsIndexCache(ctx context.Context, lang string) error {
	redisClient, err := s.container.Redis()
	if err != nil {
		return err
	}

	repository, err := s.container.ItemsRepository()
	if err != nil {
		return err
	}

	key := "GO_TOPBRANDSLIST_3_" + lang

	var cache BrandsCache

	options := items.ListOptions{
		Language: lang,
		Fields: items.ListFields{
			NameOnly:            true,
			DescendantsCount:    true,
			NewDescendantsCount: true,
		},
		TypeID:     []items.ItemType{items.BRAND},
		Limit:      items.TopBrandsCount,
		OrderBy:    []exp.OrderedExpression{goqu.C("descendants_count").Desc()},
		SortByName: true,
	}

	list, _, err := repository.List(ctx, options, false)
	if err != nil {
		return err
	}

	count, err := repository.Count(ctx, options)
	if err != nil {
		return err
	}

	cache.Items = list
	cache.Total = count

	cacheBytes, err := json.Marshal(cache) //nolint: musttag
	if err != nil {
		return err
	}

	err = redisClient.Set(ctx, key, string(cacheBytes), 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func (s *Application) generateTwinsIndexCache(ctx context.Context, lang string) error {
	var err error

	redisClient, err := s.container.Redis()
	if err != nil {
		return err
	}

	repository, err := s.container.ItemsRepository()
	if err != nil {
		return err
	}

	key := "GO_TWINS_5_" + lang

	twinsData := struct {
		Count int
		Res   []items.Item
	}{
		0,
		nil,
	}

	twinsData.Res, _, err = repository.List(ctx, items.ListOptions{
		Language: lang,
		Fields: items.ListFields{
			NameOnly: true,
		},
		DescendantItems: &items.ListOptions{
			ParentItems: &items.ListOptions{
				TypeID: []items.ItemType{items.TWINS},
				Fields: items.ListFields{
					ItemsCount:    true,
					NewItemsCount: true,
				},
			},
		},
		TypeID:  []items.ItemType{items.BRAND},
		Limit:   items.TopTwinsBrandsCount,
		OrderBy: []exp.OrderedExpression{goqu.C("items_count").Desc()},
	}, false)
	if err != nil {
		return err
	}

	twinsData.Count, err = repository.CountDistinct(ctx, items.ListOptions{
		DescendantItems: &items.ListOptions{
			ParentItems: &items.ListOptions{
				TypeID: []items.ItemType{items.TWINS},
			},
		},
		TypeID: []items.ItemType{items.BRAND},
	})
	if err != nil {
		return err
	}

	cacheBytes, err := json.Marshal(twinsData) //nolint: musttag
	if err != nil {
		return err
	}

	err = redisClient.Set(ctx, key, string(cacheBytes), 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func (s *Application) GenerateIndexCache(ctx context.Context) error {
	for lang := range s.container.Config().Languages {
		logrus.Infof("generate index cache for `%s`", lang)

		// brands
		err := s.generateBrandsIndexCache(ctx, lang)
		if err != nil {
			return err
		}

		// twins
		err = s.generateTwinsIndexCache(ctx, lang)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Application) ExportUsersToKeycloak(ctx context.Context) error {
	ur, err := s.container.UsersRepository()
	if err != nil {
		return err
	}

	return ur.ExportUsersToKeycloak(ctx)
}

func (s *Application) ListenMonitoringAMQP(ctx context.Context, quit chan bool) error {
	traffic, err := s.container.Traffic()
	if err != nil {
		return err
	}

	cfg := s.container.Config()

	logrus.Info("Monitoring listener started")

	err = traffic.Monitoring.Listen(ctx, cfg.RabbitMQ, cfg.MonitoringQueue, quit)
	if err != nil {
		logrus.Error(err.Error())

		return err
	}

	logrus.Info("Monitoring listener stopped")

	return nil
}

func (s *Application) ImageStorageGetImage(ctx context.Context, imageID int) (*APIImage, error) {
	is, err := s.container.ImageStorage()
	if err != nil {
		return nil, err
	}

	img, err := is.Image(ctx, imageID)
	if err != nil {
		return nil, err
	}

	return APIImageToGRPC(img), nil
}

func (s *Application) ImageStorageGetFormattedImage(
	ctx context.Context,
	imageID int,
	format string,
) (*APIImage, error) {
	is, err := s.container.ImageStorage()
	if err != nil {
		return nil, err
	}

	img, err := is.FormattedImage(ctx, imageID, format)
	if err != nil {
		return nil, err
	}

	return APIImageToGRPC(img), nil
}

func (s *Application) ImageStorageListBrokenImages(ctx context.Context, dir string) error {
	is, err := s.container.ImageStorage()
	if err != nil {
		return err
	}

	return is.ListBrokenImages(ctx, dir)
}

func (s *Application) ImageStorageListUnlinkedObjects(ctx context.Context, dir string) error {
	is, err := s.container.ImageStorage()
	if err != nil {
		return err
	}

	return is.ListUnlinkedObjects(ctx, dir)
}
