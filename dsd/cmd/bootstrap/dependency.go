package bootstrap

import (
	"time"

	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"gitlab.sicepat.tech/pka/sds/configs"
	"gitlab.sicepat.tech/pka/sds/internal/adapter/inbound/http"
	httpHandler "gitlab.sicepat.tech/pka/sds/internal/adapter/inbound/http/handler"
	"gitlab.sicepat.tech/pka/sds/internal/adapter/inbound/migrate"
	"gitlab.sicepat.tech/pka/sds/internal/adapter/inbound/subscriber"
	subscriberHandler "gitlab.sicepat.tech/pka/sds/internal/adapter/inbound/subscriber/handler"
	"gitlab.sicepat.tech/pka/sds/internal/adapter/inbound/worker"
	"gitlab.sicepat.tech/pka/sds/internal/adapter/outbound/dlock"
	"gitlab.sicepat.tech/pka/sds/internal/adapter/outbound/idempotency"
	"github.com/sirupsen/logrus"

	"github.com/redhajuanda/komon/logger"
	outboundkafka "gitlab.sicepat.tech/pka/sds/internal/adapter/outbound/kafka"
	"gitlab.sicepat.tech/pka/sds/internal/adapter/outbound/mariadb"
	"gitlab.sicepat.tech/pka/sds/internal/adapter/outbound/redis"
	outboundredisstream "gitlab.sicepat.tech/pka/sds/internal/adapter/outbound/redisstream"
	"gitlab.sicepat.tech/pka/sds/internal/core/port/outbound"
	"gitlab.sicepat.tech/pka/sds/internal/core/service/note"
)

// There are 4 types of resources:
// - Resource[T] is a generic resource that can be initialized and retrieved
// - ResourceRunnable[T] is a resource that can be run, this resource should implement `OnStart(ctx context.Context) error` and `OnStop(ctx context.Context) error` methods
// - ResourceExecutable[T] is a resource that can be executed, this resource should implement `Execute(ctx context.Context) error` method
// - ResourceClosable[T] is a resource that can be closed, this resource should implement `Close() error` method
type Dependency struct {
	cfgFile string

	cfg                Resource[*configs.Config]
	log                Resource[logger.Logger]
	migrators          Resource[[]migrate.Migrator]
	repository         Resource[outbound.Repository]
	serviceNote        Resource[*note.Service]
	httpHandlers       Resource[[]http.Handler]
	subscriberHandlers Resource[[]subscriber.Handler]

	qweryMain   ResourceClosable[*mariadb.Qwery]
	qweryWorker ResourceClosable[*mariadb.Qwery]
	redis       ResourceClosable[*redis.Redis]
	dlocker     ResourceClosable[*dlock.DLock]
	idempotency ResourceClosable[*idempotency.Idempotency]

	publisherRedisstream ResourceClosable[*redisstream.Publisher]
	publisherKafka       ResourceClosable[*kafka.Publisher]
	publishers           Resource[outbound.Publishers]

	subscriberRedisstream ResourceClosable[*redisstream.Subscriber]
	subscriberKafka       ResourceClosable[*kafka.Subscriber]

	migrate           ResourceExecutable[*migrate.Migrate]
	migrateGenerate   ResourceExecutable[*migrate.Generate]
	http              ResourceRunnable[*http.HTTP]
	workerOutboxRelay ResourceExecutable[*worker.WorkerOutboxRelay]
	workerGenerateID  ResourceExecutable[*worker.WorkerGenerateID]
	subscriber        ResourceRunnable[*subscriber.Subscriber]
}

// NewDependency creates a new dependency instance
func NewDependency(cfgFile string) *Dependency {
	return &Dependency{
		cfgFile: cfgFile,
	}
}

// GetConfig resolves and returns the config dependency
func (d *Dependency) GetConfig() *configs.Config {
	return d.cfg.Resolve(func() *configs.Config {
		return configs.LoadConfig(d.cfgFile)
	})
}

// GetLogger resolves and returns the logger dependency
func (d *Dependency) GetLogger() logger.Logger {
	return d.log.Resolve(func() logger.Logger {
		cfg := d.GetConfig()
		log := logger.New(cfg.App.Name, logger.Options{
			RedactedFields: cfg.Log.RedactedFields,
		})
		logger.SetLevel(logrus.Level(cfg.Log.Level))
		if cfg.Log.Format == "json" {
			logger.SetFormatter(&logrus.JSONFormatter{
				PrettyPrint: false,
			})
		} else {
			logger.SetFormatter(&logrus.TextFormatter{
				FullTimestamp: true,
				ForceColors:   true,
			})
		}
		return log.WithParam("service", cfg.App.Name)
	})
}

// GetRedis resolves and returns the redis dependency
func (d *Dependency) GetRedis() *redis.Redis {
	return d.redis.Resolve(func() *redis.Redis {
		cfg := d.GetConfig()
		return redis.New(
			redis.Param{
				Sentinel:     cfg.Cache.Redis.Sentinel,
				MasterName:   cfg.Cache.Redis.MasterName,
				Username:     cfg.Cache.Redis.Username,
				Password:     cfg.Cache.Redis.Password,
				Hosts:        cfg.Cache.Redis.Hosts,
				DB:           cfg.Cache.Redis.DB,
				MinIdleConns: cfg.Cache.Redis.MinIdleConns,
				PoolSize:     cfg.Cache.Redis.PoolSize,
			},
			d.GetLogger())
	})
}

// GetQweryMain resolves and returns the qwery main dependency
func (d *Dependency) GetQweryMain() *mariadb.Qwery {
	return d.qweryMain.Resolve(func() *mariadb.Qwery {
		cfg := d.GetConfig()
		return mariadb.NewQwery(
			mariadb.ParamQwery{
				Username:        cfg.Database.MariaDBMain.Username,
				Password:        cfg.Database.MariaDBMain.Password,
				Host:            cfg.Database.MariaDBMain.Host,
				Port:            cfg.Database.MariaDBMain.Port,
				DBName:          cfg.Database.MariaDBMain.DBName,
				MaxOpenConns:    cfg.Database.MariaDBMain.MaxOpenConns,
				MaxIdleConns:    cfg.Database.MariaDBMain.MaxIdleConns,
				ConnMaxLifetime: cfg.Database.MariaDBMain.ConnMaxLifetime,
				ConnMaxIdleTime: cfg.Database.MariaDBMain.ConnMaxIdleTime,
			},
			d.GetLogger(),
		)
	})
}

// GetQweryWorker resolves and returns the qwery worker dependency
func (d *Dependency) GetQweryWorker() *mariadb.Qwery {
	return d.qweryWorker.Resolve(func() *mariadb.Qwery {
		cfg := d.GetConfig()
		return mariadb.NewQwery(
			mariadb.ParamQwery{
				Username:        cfg.Database.MariaDBWorker.Username,
				Password:        cfg.Database.MariaDBWorker.Password,
				Host:            cfg.Database.MariaDBWorker.Host,
				Port:            cfg.Database.MariaDBWorker.Port,
				DBName:          cfg.Database.MariaDBWorker.DBName,
				MaxOpenConns:    cfg.Database.MariaDBWorker.MaxOpenConns,
				MaxIdleConns:    cfg.Database.MariaDBWorker.MaxIdleConns,
				ConnMaxLifetime: cfg.Database.MariaDBWorker.ConnMaxLifetime,
				ConnMaxIdleTime: cfg.Database.MariaDBWorker.ConnMaxIdleTime,
			},
			d.GetLogger(),
		)
	})
}

// GetRepository resolves and returns the repository dependency
func (d *Dependency) GetRepository(qwery *mariadb.Qwery) outbound.Repository {
	return d.repository.Resolve(func() outbound.Repository {
		return mariadb.NewMariaDBRepository(d.GetConfig(), d.GetLogger(), qwery, d.GetPublishers())
	})
}

// GetPublisherRedisstream resolves and returns the publisher dependency
func (d *Dependency) GetPublisherRedisstream() *redisstream.Publisher {
	return d.publisherRedisstream.Resolve(func() *redisstream.Publisher {
		cfg := d.GetConfig()
		param := outboundredisstream.ParamPublisher{
			ParamRedis: outboundredisstream.ParamRedis{
				Sentinel:     cfg.Cache.Redis.Sentinel,
				MasterName:   cfg.Cache.Redis.MasterName,
				Username:     cfg.Cache.Redis.Username,
				Password:     cfg.Cache.Redis.Password,
				Hosts:        cfg.Cache.Redis.Hosts,
				DB:           cfg.Cache.Redis.DB,
				MinIdleConns: cfg.Cache.Redis.MinIdleConns,
				PoolSize:     cfg.Cache.Redis.PoolSize,
			},
			DefaultMaxlen: cfg.Event.Redisstream.Publisher.DefaultMaxlen,
		}
		return outboundredisstream.NewPublisher(param)
	})
}

// GetPublisherKafka resolves and returns the publisher dependency
func (d *Dependency) GetPublisherKafka() *kafka.Publisher {
	return d.publisherKafka.Resolve(func() *kafka.Publisher {
		cfg := d.GetConfig()
		param := outboundkafka.ParamPublisher{
			Brokers:      cfg.Event.Kafka.Publisher.Brokers,
			DebugEnabled: cfg.Event.Kafka.Publisher.DebugEnabled,
			TraceEnabled: cfg.Event.Kafka.Publisher.TraceEnabled,
		}
		return outboundkafka.NewPublisher(param, d.GetLogger())
	})
}

func (d *Dependency) GetPublishers() outbound.Publishers {
	return d.publishers.Resolve(func() outbound.Publishers {
		publishers := outbound.Publishers{}
		cfg := d.GetConfig()
		if cfg.Event.Redisstream.Publisher.Enabled {
			publishers[outbound.PublisherTargetRedisstream] = d.GetPublisherRedisstream()
		}
		if cfg.Event.Kafka.Publisher.Enabled {
			publishers[outbound.PublisherTargetKafka] = d.GetPublisherKafka()
		}
		return publishers
	})
}

// GetSubscriber resolves and returns the subscriber dependency
func (d *Dependency) GetSubscriberRedisstream(subscriberID string) *redisstream.Subscriber {
	return d.subscriberRedisstream.Resolve(func() *redisstream.Subscriber {

		cfg := d.GetConfig()
		cfgSubscriber := cfg.Event.Redisstream.Subscribers.GetByID(subscriberID)
		if cfgSubscriber == nil {
			d.GetLogger().Fatalf("redisstream subscriber config not found for subscriber ID: %s", subscriberID)
		}
		param := outboundredisstream.ParamSubscriber{
			ParamRedis: outboundredisstream.ParamRedis{
				Sentinel:     cfg.Cache.Redis.Sentinel,
				MasterName:   cfg.Cache.Redis.MasterName,
				Username:     cfg.Cache.Redis.Username,
				Password:     cfg.Cache.Redis.Password,
				Hosts:        cfg.Cache.Redis.Hosts,
				DB:           cfg.Cache.Redis.DB,
				MinIdleConns: cfg.Cache.Redis.MinIdleConns,
				PoolSize:     cfg.Cache.Redis.PoolSize,
			},
			ConsumerGroup: cfgSubscriber.ConsumerGroup,
		}
		return outboundredisstream.NewSubscriber(param)
	})
}

// GetSubscriberKafka resolves and returns the subscriber dependency
func (d *Dependency) GetSubscriberKafka(subscriberID string) *kafka.Subscriber {
	return d.subscriberKafka.Resolve(func() *kafka.Subscriber {
		cfg := d.GetConfig()
		cfgSubscriber := cfg.Event.Kafka.Subscribers.GetByID(subscriberID)
		if cfgSubscriber == nil {
			d.GetLogger().Fatalf("kafka subscriber config not found for subscriber ID: %s", subscriberID)
		}
		param := outboundkafka.ParamSubscriber{
			Brokers:       cfgSubscriber.Brokers,
			ConsumerGroup: cfgSubscriber.ConsumerGroup,
			DebugEnabled:  cfgSubscriber.DebugEnabled,
			TraceEnabled:  cfgSubscriber.TraceEnabled,
		}
		return outboundkafka.NewSubscriber(param, d.GetLogger())
	})
}

// GetIdempotency resolves and returns the idempotency dependency
func (d *Dependency) GetIdempotency() *idempotency.Idempotency {
	return d.idempotency.Resolve(func() *idempotency.Idempotency {
		cfg := d.GetConfig()
		return idempotency.NewIdempotency(idempotency.Param{
			Sentinel:     cfg.Cache.Redis.Sentinel,
			MasterName:   cfg.Cache.Redis.MasterName,
			Username:     cfg.Cache.Redis.Username,
			Password:     cfg.Cache.Redis.Password,
			Hosts:        cfg.Cache.Redis.Hosts,
			DB:           cfg.Cache.Redis.DB,
			MinIdleConns: cfg.Cache.Redis.MinIdleConns,
			PoolSize:     cfg.Cache.Redis.PoolSize,
		}, d.GetLogger())
	})
}

// GetDLocker resolves and returns the dlocker dependency
func (d *Dependency) GetDLocker() *dlock.DLock {
	return d.dlocker.Resolve(func() *dlock.DLock {
		cfg := d.GetConfig()
		param := dlock.Param{
			Sentinel:     cfg.Cache.Redis.Sentinel,
			MasterName:   cfg.Cache.Redis.MasterName,
			Username:     cfg.Cache.Redis.Username,
			Password:     cfg.Cache.Redis.Password,
			Hosts:        cfg.Cache.Redis.Hosts,
			DB:           cfg.Cache.Redis.DB,
			MinIdleConns: cfg.Cache.Redis.MinIdleConns,
			PoolSize:     cfg.Cache.Redis.PoolSize,
		}
		return dlock.New(param, d.GetLogger())
	})
}

// GetMigrators resolves and returns the migrators dependency
func (d *Dependency) GetMigrators() []migrate.Migrator {
	return d.migrators.Resolve(func() []migrate.Migrator {
		return []migrate.Migrator{
			mariadb.NewMigrator(d.GetConfig(), d.GetLogger(), d.GetQweryMain()),
		}
	})
}

// GetServiceNote resolves and returns the service note dependency
func (d *Dependency) GetServiceNote(repo outbound.Repository) *note.Service {
	return d.serviceNote.Resolve(func() *note.Service {
		return note.NewService(d.GetConfig(), d.GetLogger(), repo, d.GetRedis())
	})
}

// GetHTTPHandlers resolves and returns the http handlers dependency
func (d *Dependency) GetHTTPHandlers() []http.Handler {
	return d.httpHandlers.Resolve(func() []http.Handler {
		repo := d.GetRepository(d.GetQweryMain())
		return []http.Handler{
			httpHandler.NewNoteHandler(d.GetConfig(), d.GetLogger(), d.GetServiceNote(repo)),
		}
	})
}

// GetHTTP resolves and returns the http dependency
func (d *Dependency) GetHTTP() *http.HTTP {
	return d.http.Resolve(func() *http.HTTP {
		return http.NewHTTP(d.GetConfig(), d.GetLogger(), d.GetHTTPHandlers())
	})
}

// GetSubscriberHandlers resolves and returns the subscriber handlers dependency
func (d *Dependency) GetSubscriberHandlers() []subscriber.Handler {
	return d.subscriberHandlers.Resolve(func() []subscriber.Handler {
		return []subscriber.Handler{
			subscriberHandler.NewNoteHandler(d.GetConfig(), d.GetLogger(), d.GetSubscriberKafka("general")),
		}
	})
}

// GetSubscriber resolves and returns the subscriber dependency
func (d *Dependency) GetSubscriber(closeTimeout time.Duration) *subscriber.Subscriber {
	return d.subscriber.Resolve(func() *subscriber.Subscriber {
		return subscriber.NewSubscriber(d.GetConfig(), d.GetLogger(), d.GetIdempotency(), d.GetSubscriberHandlers(), closeTimeout)
	})
}

// GetMigrate resolve migrate dependency instance
func (d *Dependency) GetMigrate(migrateType string, max int, repository string) *migrate.Migrate {
	return d.migrate.Resolve(func() *migrate.Migrate {
		return migrate.NewMigrate(
			d.GetConfig(),
			d.GetLogger(),
			d.GetMigrators(),
			migrate.MigrateParams{
				MigrationType: migrateType,
				Max:           max,
				Repository:    repository,
			})
	})
}

// GetMigrateGenerate resolves and returns the migrate generate dependency
func (d *Dependency) GetMigrateGenerate(repository string, fileName string) *migrate.Generate {
	return d.migrateGenerate.Resolve(func() *migrate.Generate {
		return migrate.NewGenerate(
			d.GetConfig(),
			d.GetLogger(),
			d.GetMigrators(),
			migrate.GenerateParams{
				Repository: repository,
				FileName:   fileName,
			})
	})
}

// GetWorkerGenerateID resolves and returns the worker generate id dependency
func (d *Dependency) GetWorkerGenerateID() *worker.WorkerGenerateID {
	return d.workerGenerateID.Resolve(func() *worker.WorkerGenerateID {
		return worker.NewWorkerGenerateID(d.GetConfig(), d.GetLogger())
	})
}

// GetWorkerOutboxRelay resolves and returns the worker relay outbox dependency
func (d *Dependency) GetWorkerOutboxRelay() *worker.WorkerOutboxRelay {
	return d.workerOutboxRelay.Resolve(func() *worker.WorkerOutboxRelay {
		repo := d.GetRepository(d.GetQweryWorker())
		relayOutbox := worker.NewWorkerOutboxRelay(d.GetConfig(), d.GetLogger(), repo)
		return relayOutbox
	})
}