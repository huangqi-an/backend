package redisclient

import (
	"context"
	"jwt-redis/config"
	"time"

	"github.com/redis/go-redis/v9"
)

// UniversalClient 让同一份代码既能连单机也能连 Cluster
/**
返回值是 redis.UniversalClient
这是 go-redis/v9 提供的一个接口，同时被下面三种类型实现：

类型	场景
*redis.Client	单机
*redis.ClusterClient	集群
*redis.Ring	分片
*/
func New(cfg *config.Config) (redis.UniversalClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDb, //逻辑库编号，Cluster 模式下必须为 0
		PoolSize:     20,
		MinIdleConns: 5,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}
	return rdb, nil
}
