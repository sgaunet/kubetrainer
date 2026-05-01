// Package database provides a PostgreSQL-backed implementation of the
// kubetrainer storage layer including connection helpers and migrations.
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	// Register the dbmate postgres driver.
	_ "github.com/amacneil/dbmate/v2/pkg/driver/postgres"
	// Register the lib/pq driver as the database/sql postgres driver.
	_ "github.com/lib/pq"
	"github.com/sgaunet/dsn/v2/pkg/dsn"
)

//go:embed db/migrations/*.sql
var fs embed.FS

const (
	waitForDBTimeout    = 30 * time.Second
	dbReadyPollInterval = 1 * time.Second
	defaultPostgresPort = "5432"
)

// Database describes the storage operations required by kubetrainer.
type Database interface {
	IsConnected() bool
	GetDB() *sql.DB
	Close() error
}

// Postgres is the PostgreSQL implementation of Database.
type Postgres struct {
	DB               *sql.DB
	pgDataSourceName dsn.DSN
}

// NewPostgres opens a PostgreSQL connection from a DSN and verifies it.
func NewPostgres(pgdsn string) (*Postgres, error) {
	d, err := dsn.New(pgdsn)
	if err != nil {
		return nil, fmt.Errorf("parsing postgres dsn: %w", err)
	}
	p := &Postgres{pgDataSourceName: d}
	fmt.Println("Connecting to database...", d.GetPostgresUri())
	db, err := sql.Open("postgres", d.GetPostgresUri())
	if err != nil {
		return nil, fmt.Errorf("opening postgres connection: %w", err)
	}
	p.DB = db
	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}
	return p, nil
}

// WaitForDB waits for the database to be ready.
func WaitForDB(ctx context.Context, pgdsn string) error {
	d, err := dsn.New(pgdsn)
	if err != nil {
		return fmt.Errorf("parsing postgres dsn: %w", err)
	}
	chDBReady := make(chan struct{})
	go func() {
		for {
			db, err := sql.Open("postgres", d.GetPostgresUri())
			select {
			case <-ctx.Done():
				return
			default:
				if err == nil {
					err = db.PingContext(ctx)
					if err == nil {
						_ = db.Close()
						close(chDBReady)
						return
					}
					_ = db.Close()
					fmt.Println("Database not ready (not pingable)")
					time.Sleep(dbReadyPollInterval)
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("waiting for database: %w", ctx.Err())
	case <-chDBReady:
		return nil
	}
}

// InitDB applies pending database migrations.
func (p *Postgres) InitDB() error {
	u, err := url.Parse(genDbmateURI(p.pgDataSourceName))
	if err != nil {
		return fmt.Errorf("parsing dbmate url: %w", err)
	}
	db := dbmate.New(u)
	db.FS = fs
	db.AutoDumpSchema = false

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(waitForDBTimeout))
	defer cancel()
	if err := WaitForDB(ctx, p.pgDataSourceName.String()); err != nil {
		return err
	}
	fmt.Println("Migrations:")
	migrations, err := db.FindMigrations()
	if err != nil {
		return fmt.Errorf("finding migrations: %w", err)
	}
	for _, m := range migrations {
		fmt.Println(m.Version, m.FilePath)
	}
	fmt.Println("\nApplying...")
	if err := db.CreateAndMigrate(); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	return nil
}

// Close releases the underlying database resources.
func (p *Postgres) Close() error {
	if err := p.DB.Close(); err != nil {
		return fmt.Errorf("closing postgres connection: %w", err)
	}
	return nil
}

// GetDB returns the underlying sql.DB handle.
func (p *Postgres) GetDB() *sql.DB {
	return p.DB
}

// IsConnected reports whether the database connection is currently usable.
func (p *Postgres) IsConnected() bool {
	if p.DB == nil {
		return false
	}
	err := p.DB.PingContext(context.Background())
	return err == nil
}

func genDbmateURI(d dsn.DSN) string {
	hostPort := net.JoinHostPort(d.GetHost(), d.GetPort(defaultPostgresPort))
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		d.GetUser(),
		d.GetPassword(),
		hostPort,
		d.GetDBName(),
		d.GetParameter("sslmode"))
}
