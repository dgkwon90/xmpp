package memstore

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"log"
	"time"
)

type RedisSingle struct {
	client *redis.Client
}

func (r RedisSingle) New(client *redis.Client) *RedisSingle {
	return &RedisSingle{client: client}
}

func (r *RedisSingle) Close() error {
	if r.client != nil {
		log.Printf("RedisSingle Close\n")
		err := r.client.Close()
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

func (r *RedisSingle) Connect() error {
	ctx := context.Background()
	result, err := r.client.Ping(ctx).Result()
	if err != nil {
		return err
	}
	log.Printf("Connect: %v", result)
	return nil
}

func (r *RedisSingle) SetData(key string, val interface{}, duration time.Duration) error {
	ctx := context.Background()
	err := r.client.Set(ctx, key, val, duration).Err()
	if err != nil {
		return err
	}
	log.Printf("SetData: %v=%v", key, val)
	return nil
}

func (r *RedisSingle) GetData(key string) (string, error) {
	ctx := context.Background()
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	log.Printf("GetData: %v=%v", key, val)
	return val, nil
}

func (r *RedisSingle) GetData2(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	successCh := make(chan string)
	errorCh := make(chan error)

	go func(successCh chan string, errorCh chan error) {
		result, err := r.client.Get(ctx, key).Result()
		if err != nil {
			errorCh <- err
			return
		}
		successCh <- result
	}(successCh, errorCh)

	select {
	case <-ctx.Done():
		return "", ctx.Err()

	case result := <-successCh:
		return result, nil

	case result := <-errorCh:
		return "", result
	}
}

func (r *RedisSingle) DelData(key string) error {
	ctx := context.Background()
	_, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisSingle) ExpireData(key string, expiration time.Duration) error {
	ctx := context.Background()
	result, err := r.client.Expire(ctx, key, expiration).Result()
	if err != nil {
		return err
	}
	log.Printf("ExpireData: %v(%v)", key, result)
	return nil
}

func (r *RedisSingle) ExpireGTData(key string, expiration time.Duration) error {
	ctx := context.Background()
	result, err := r.client.ExpireGT(ctx, key, expiration).Result()
	if err != nil {
		return err
	}
	log.Printf("ExpireGTData: %v(%v)", key, result)
	return nil
}

func (r *RedisSingle) ExistsKey(key string) (int64, error) {
	ctx := context.Background()
	result, err := r.client.Exists(ctx, key).Result()
	if err == redis.Nil {
		log.Println("key does not exist")
		return result, err
	} else if err != nil {
		return result, err
	}
	log.Printf("ExistsKey: %v=%v", key, result)
	return result, nil
}

func (r *RedisSingle) SAdd(key string, members ...interface{}) error {
	ctx := context.Background()
	count, err := r.client.SAdd(ctx, key, members).Result()
	if err != nil {
		return err
	}
	log.Printf("SAdd: %v(%v)", key, count)
	return nil
}

func (r *RedisSingle) SRem(key string, members ...interface{}) error {
	ctx := context.Background()
	count, err := r.client.SRem(ctx, key, members).Result()
	if err != nil {
		return err
	}
	log.Printf("SRem: %v(%v)", key, count)
	return nil
}

func (r *RedisSingle) SMIsMember(key string, members ...interface{}) (map[string]bool, error) {
	ctx := context.Background()
	existsMembers := make(map[string]bool)
	resultExists, err := r.client.SMIsMember(ctx, key, members).Result()
	if err != nil {
		return nil, err
	}

	for i, member := range members {
		existsMembers[member.(string)] = resultExists[i]
	}

	log.Printf("SMIsMember key: %v, members: %v\n", key, existsMembers)
	return existsMembers, nil
}

func (r *RedisSingle) SMembers(key string) ([]string, error) {
	ctx := context.Background()
	members, err := r.client.SMembers(ctx, key).Result()
	if err == redis.Nil {
		return nil, errors.New("key does not exist")
	} else if err != nil {
		return nil, err
	}

	log.Printf("SMembers key: %v, values: %v\n", key, members)
	return members, err
}

func (r *RedisSingle) SCard(key string) (int64, error) {
	ctx := context.Background()
	val, err := r.client.SCard(ctx, key).Result()
	if err != nil {
		return val, err
	}
	log.Printf("SCard key: %v, val: %v\n", key, val)
	return val, nil
}
