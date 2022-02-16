package cache

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"time"
)

var (
	RedisClient *redis.Client
	ctx         = context.Background() // 最新版本的redis需要传上下文参数
)

// Config redis 配置
type Config struct {
	Addr         string
	Password     string
	DB           int
	MinIdleConn  int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	PoolTimeout  time.Duration
}

// Init 初始化redis连接池
func Init(c *Config) (err error) {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:         c.Addr,
		Password:     c.Password,
		DB:           c.DB,
		MinIdleConns: c.MinIdleConn,
		DialTimeout:  c.DialTimeout,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,
		PoolSize:     c.PoolSize,
		PoolTimeout:  c.PoolTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = RedisClient.Ping(ctx).Result()
	return
}

// Set 存string
func Set(key string, value interface{}) (err error) {
	err = RedisClient.Set(ctx, key, value, 0).Err()
	if err != nil {
		err = errors.New("Fail to cache data, err: " + err.Error())
		return
	}
	return
}

// Get 取string
func Get(key string) (string, error) {
	return RedisClient.Get(ctx, key).Result()
}

// Exists 判断缓存项是否存在
func Exists(key string) (res bool, err error) {
	result, err := RedisClient.Exists(ctx, key).Result()
	if err != nil {
		err = errors.New("Fail to determine whether the element exists, err: " + err.Error())
		return
	}
	if result > 0 {
		return true, nil
	} else {
		return false, nil
	}
}
