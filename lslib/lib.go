package lslib

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	"github.com/supersupersimple/litestream-lib"
	lss3 "github.com/supersupersimple/litestream-lib/s3"
)

const (
	EnvLitestreamUrl             = "LITESTREAM_URL"
	EnvLitestreamAccessKeyID     = "LITESTREAM_ACCESS_KEY_ID"
	EnvLitestreamSecretAccessKey = "LITESTREAM_SECRET_ACCESS_KEY"
)

type DB interface {
	Open(ctx context.Context) (db *sql.DB, err error)
	Close(ctx context.Context)
}

type Config struct {
	Dsn string

	LitestreamUrl string

	DriverName string
}

func NewConfig(dsn string) *Config {
	url := os.Getenv(EnvLitestreamUrl)
	return &Config{
		Dsn:           dsn,
		LitestreamUrl: url,
		DriverName:    "sqlite3",
	}
}

func (c *Config) WithLsUrl(lsUrl string) *Config {
	c.LitestreamUrl = lsUrl
	return c
}

func (c *Config) WithDriverName(driverName string) *Config {
	c.DriverName = driverName
	return c
}

type impl struct {
	dsn           string
	litestreamUrl string

	driverName string

	db   *sql.DB
	lsdb *litestream.DB
}

func NewDB(conf *Config) DB {
	return &impl{
		dsn:           conf.Dsn,
		litestreamUrl: conf.LitestreamUrl,
		driverName:    conf.DriverName,
	}
}

func (i *impl) Open(ctx context.Context) (db *sql.DB, err error) {
	// start litestream replicate
	if i.litestreamUrl != "" {
		i.lsdb, err = i.replicate(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Open database file and return
	db, err = sql.Open(i.driverName, i.dsn)
	return db, err
}

func (i *impl) Close(ctx context.Context) {
	i.db.Close()
	i.lsdb.SoftClose(ctx)
}

func (i *impl) replicate(ctx context.Context) (*litestream.DB, error) {
	// Create Litestream DB reference for managing replication.
	lsdb := litestream.NewDB(i.dsn)
	lsdb.SetDriverName(i.driverName)

	// Build S3 replica and attach to database.
	client := lss3.NewReplicaClient()
	_, host, path, err := ParseReplicaURL(i.litestreamUrl)
	if err != nil {
		return nil, err
	}
	client.Path = path
	client.Bucket, client.Region, client.Endpoint, client.ForcePathStyle = lss3.ParseHost(host)
	client.AccessKeyID = os.Getenv(EnvLitestreamAccessKeyID)
	client.SecretAccessKey = os.Getenv(EnvLitestreamSecretAccessKey)

	replica := litestream.NewReplica(lsdb, "s3")
	replica.Client = client

	lsdb.Replicas = append(lsdb.Replicas, replica)

	if err := i.restore(ctx, replica); err != nil {
		return nil, err
	}

	// Initialize database.
	if err := lsdb.Open(); err != nil {
		return nil, err
	}

	return lsdb, nil
}

func (i *impl) restore(ctx context.Context, replica *litestream.Replica) (err error) {
	// Skip restore if local database already exists.
	if _, err := os.Stat(replica.DB().Path()); err == nil {
		slog.Info("local database already exists, skipping restore")
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	// Configure restore to write out to DSN path.
	opt := litestream.NewRestoreOptions()
	opt.OutputPath = replica.DB().Path()
	opt.DriverName = i.driverName

	// Determine the latest generation to restore from.
	if opt.Generation, _, err = replica.CalcRestoreTarget(ctx, opt); err != nil {
		return err
	}

	// Only restore if there is a generation available on the replica.
	// Otherwise we'll let the application create a new database.
	if opt.Generation == "" {
		slog.Info("no generation found, creating new database")
		return nil
	}

	slog.Info("restoring replica for generation", "generation", opt.Generation)
	if err := replica.Restore(ctx, opt); err != nil {
		return err
	}
	slog.Info("restore complete")
	return nil
}
