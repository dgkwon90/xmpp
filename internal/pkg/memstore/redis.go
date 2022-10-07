package memstore

import (
	"fmt"
	"github.com/go-redis/redis/v8"
)

func NewClient(config *Config) Redis {
	if len(config.Clusters) <= 0 {
		//single
		redisClient := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%v:%v", config.Address.Host, config.Address.Port),
			Password: config.Password, // password
			DB:       0,               // namespace
		})
		return RedisSingle{}.New(redisClient)
	} else {
		//cluster
		configSlots := ParseRedisConfig(config.Clusters)
		return RedisCluster{}.New(configSlots, &config.Password)
	}
}
