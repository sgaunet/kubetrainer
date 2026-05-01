// Package main provides the kubetrainer entry point.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sgaunet/dsn/v2/pkg/dsn"
	"github.com/sgaunet/kubetrainer/internal/database"
	"github.com/sgaunet/kubetrainer/pkg/config"
)

const initDBTimeout = 30 * time.Second

// initDB initializes the database connection and creates the necessary tables.
func initDB(cfgApp *config.Config) (*database.Postgres, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(initDBTimeout))
	defer cancel()
	if err := database.WaitForDB(ctx, cfgApp.DBCfg.DbDSN); err != nil {
		return nil, fmt.Errorf("waiting for database: %w", err)
	}
	pg, err := database.NewPostgres(cfgApp.DBCfg.DbDSN)
	if err != nil {
		return nil, fmt.Errorf("creating postgres client: %w", err)
	}
	if err := pg.InitDB(); err != nil {
		return nil, fmt.Errorf("initializing database: %w", err)
	}
	return pg, nil
}

// initRedisConnection initializes the redis connection.
func initRedisConnection(redisdsn string) (*redis.Client, error) {
	d, err := dsn.New(redisdsn)
	if err != nil {
		return nil, fmt.Errorf("parsing redis dsn: %w", err)
	}
	addr := fmt.Sprintf("%s:%s", d.GetHost(), d.GetPort("6379"))
	redisClient := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: d.GetUser(),
		Password: d.GetPassword(),
	})
	if _, err := redisClient.Ping(context.TODO()).Result(); err != nil {
		return nil, fmt.Errorf("pinging redis: %w", err)
	}
	return redisClient, nil
}
