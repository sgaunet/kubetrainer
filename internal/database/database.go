package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"time"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/postgres"
	_ "github.com/lib/pq"
	"github.com/sgaunet/dsn/v2/pkg/dsn"
)

//go:embed db/migrations/*.sql
var fs embed.FS

type Postgres struct {
	DB               *sql.DB
	pgDataSourceName dsn.DSN
}

func NewPostgres(pgdsn string) (*Postgres, error) {
	d, err := dsn.New(pgdsn)
	if err != nil {
		return nil, err
	}
	p := &Postgres{pgDataSourceName: d}
	// err = p.InitDB()
	// if err != nil {
	// 	return nil, err
	// }
	fmt.Println("Connecting to database...", d.GetPostgresUri())
	db, err := sql.Open("postgres", d.GetPostgresUri())
	if err != nil {
		return nil, err
	}
	p.DB = db
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return p, err
}

// WaitForDB waits for the database to be ready
func WaitForDB(ctx context.Context, pgdsn string) error {
	d, err := dsn.New(pgdsn)
	if err != nil {
		return err
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
					err = db.Ping()
					defer db.Close()
					if err == nil {
						close(chDBReady)
						return
					}
					fmt.Println("Database not ready (not pingable)")
					time.Sleep(1 * time.Second)
				}
				// fmt.Println("Waiting for database to be ready...", pgdsn, err.Error())
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-chDBReady:
		return nil
	}
}

func (p *Postgres) InitDB() error {
	u, err := url.Parse(genDbmateUri(p.pgDataSourceName))
	if err != nil {
		return err
	}
	fmt.Println(u.String())
	db := dbmate.New(u)
	db.FS = fs
	db.AutoDumpSchema = false

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
	err = WaitForDB(ctx, p.pgDataSourceName.String())
	defer cancel()
	if err != nil {
		return err
	}
	fmt.Println("Migrations:")
	migrations, err := db.FindMigrations()
	if err != nil {
		return err
	}
	for _, m := range migrations {
		fmt.Println(m.Version, m.FilePath)
	}
	fmt.Println("\nApplying...")
	return db.CreateAndMigrate()
}

func (p *Postgres) Close() error {
	return p.DB.Close()
}

func (p *Postgres) GetDB() *sql.DB {
	return p.DB
}

func genDbmateUri(d dsn.DSN) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.GetUser(),
		d.GetPassword(),
		d.GetHost(),
		d.GetPort("5432"),
		d.GetDBName(),
		d.GetParameter("sslmode"))
}
