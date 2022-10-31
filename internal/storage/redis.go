package storage

import (
	"github.com/go-redis/redis"
	"go.uber.org/zap"
)

// Config contains all settings for the Redis memory.
type Config struct {
	Addr     string
	Key      string
	Password string
	DB       int
	Logger   *zap.Logger
}

type RedisMemory struct {
	logger *zap.Logger
	Client *redis.Client
	hkey   string
}

func NewRedisStorage(config Config) Memory {

	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	memory := &RedisMemory{
		logger: config.Logger,
		hkey:   config.Key,
		Client: client,
	}

	return memory
}

func (rm *RedisMemory) Set(key string, value []byte) error {
	rm.logger.Info("Set", zap.String("Setting key", key))
	resp := rm.Client.HSet(rm.hkey, key, value)
	return resp.Err()
}

func (rm *RedisMemory) Get(key string) ([]byte, bool, error) {
	resp, err := rm.Client.HGet(rm.hkey, key).Result()
	switch {
	case err != nil:
		return nil, false, err
	case err == redis.Nil:
		return nil, false, nil
	default:
		return []byte(resp), true, nil
	}
}

func (rm *RedisMemory) Delete(key string) (bool, error) {
	resp, err := rm.Client.HDel(rm.hkey, key).Result()
	return resp > 0, err
}

func (b *RedisMemory) Keys() ([]string, error) {
	return b.Client.HKeys(b.hkey).Result()
}

func (b *RedisMemory) Close() error {
	return b.Client.Close()
}
