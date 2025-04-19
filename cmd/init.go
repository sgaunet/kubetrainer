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

// initDB initializes the database connection and creates the necessary tables
func initDB(cfgApp *config.Config) (*database.Postgres, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
	defer cancel()
	err := database.WaitForDB(ctx, cfgApp.DBCfg.DbDSN)
	if err != nil {
		return nil, err
	}
	pg, err := database.NewPostgres(cfgApp.DBCfg.DbDSN)
	if err != nil {
		return nil, err
	}
	err = pg.InitDB()
	return pg, err
}

// initRedisConnection initializes the redis connection
func initRedisConnection(redisdsn string) (*redis.Client, error) {
	var err error
	d, err := dsn.New(redisdsn)
	if err != nil {
		fmt.Println("Error in redis dsn", err.Error(), redisdsn)
		return nil, err
	}
	addr := fmt.Sprintf("%s:%s", d.GetHost(), d.GetPort("6379"))
	redisClient := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: d.GetUser(),
		Password: d.GetPassword(),
	})
	_, err = redisClient.Ping(context.TODO()).Result()
	return redisClient, err
}
