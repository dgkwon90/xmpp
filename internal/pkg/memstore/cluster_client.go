package memstore

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v8"
	"log"
	"net"
	"strings"
	"time"
)

type RedisCluster struct {
	client *redis.ClusterClient
	//ring    *redis.Ring
	clusterNodes []redis.ClusterSlot
}

type RedisClusterSlot struct {
	MasterAddr string ` json:"masterAddr"`
	SlaveAddr  string ` json:"slaveAddr"`
	Start      int    `json:"start"`
	End        int    `json:"end"`
}

func ParseRedisConfig(configJson string) *[]RedisClusterSlot {
	log.Printf("start ParseRedisConfig: %v", configJson)
	clusterNodes := make([]RedisClusterSlot, 0)
	err := json.Unmarshal([]byte(configJson), &clusterNodes)
	if err != nil {
		log.Printf("ParseRedisConfig: %v", err.Error())
	}
	return &clusterNodes
}

func (r RedisCluster) New(configSlots *[]RedisClusterSlot, configPassword *string) *RedisCluster {
	rc := RedisCluster{
		client: nil,
		//ring:         nil,
		clusterNodes: nil,
	}

	rc.clusterNodes = make([]redis.ClusterSlot, 0)
	for _, slot := range *configSlots {
		clusterSlot := redis.ClusterSlot{
			// First node with 1 master and 1 slave.
			Start: slot.Start,
			End:   slot.End,
			Nodes: []redis.ClusterNode{{
				Addr: slot.MasterAddr, // master
			}, {
				Addr: slot.SlaveAddr, // slave
			}},
		}
		rc.clusterNodes = append(rc.clusterNodes, clusterSlot)
	}

	clusterSlots := func(ctx context.Context) ([]redis.ClusterSlot, error) {
		return rc.clusterNodes, nil
	}

	client := redis.NewClusterClient(&redis.ClusterOptions{
		ReadOnly:       false,
		RouteByLatency: false,
		RouteRandomly:  false,
		ClusterSlots:   clusterSlots,
		Dialer:         rc.Dialer,
		Password:       *configPassword,
	})
	rc.client = client
	return &rc
}

func (r *RedisCluster) Close() error {
	if r.client != nil {
		log.Printf("RedisCluster Close\n")
		err := r.client.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RedisCluster) Dialer(ctx context.Context, network, addr string) (net.Conn, error) {
	log.Printf("Dialer Called network(%v), addr(%v) ", network, addr)
	conn, err := net.DialTimeout(network, addr, time.Millisecond*100)
	if err != nil {
		log.Printf("Dialer failed with an error : %v ", err)

		for _, v := range r.clusterNodes {
			masterAddr := v.Nodes[0].Addr
			slaveAddr := v.Nodes[1].Addr
			if strings.Compare(masterAddr, addr) == 0 {
				log.Printf("Try to Current Slot's Slave : %v", slaveAddr)
				conn, err = net.DialTimeout(network, slaveAddr, time.Millisecond*100)

				if err != nil {
					log.Printf("Slave Approch Failed : %v", err)
				}
				return conn, err
			}

			if strings.Compare(slaveAddr, addr) == 0 {
				log.Printf("Try to Current Slot's Master : %v", masterAddr)
				conn, err = net.DialTimeout(network, masterAddr, time.Millisecond*100)

				if err != nil {
					log.Printf("Master Approch Failed : %v", err)
				}
				return conn, err
			}
		}
	}

	return conn, err
}

func (r *RedisCluster) Connect() error {
	ctx := context.Background()
	result, err := r.client.Ping(ctx).Result()
	if err != nil {
		return err
	}
	log.Printf("Connect: %v", result)
	return nil
}

func (r *RedisCluster) SetData(key string, val interface{}, duration time.Duration) error {
	ctx := context.Background()
	err := r.client.Set(ctx, key, val, duration).Err()
	if err != nil {
		return err
	}
	log.Printf("SetData: %v=%v", key, val)
	return nil
}

func (r *RedisCluster) GetData(key string) (string, error) {
	ctx := context.Background()
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	log.Printf("GetData: %v=%v", key, val)
	return val, nil
}

func (r *RedisCluster) DelData(key string) error {
	ctx := context.Background()
	count, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return err
	}
	log.Printf("DelData: %v", count)
	return nil
}

func (r *RedisCluster) ExpireData(key string, expiration time.Duration) error {
	ctx := context.Background()
	result, err := r.client.Expire(ctx, key, expiration).Result()
	if err != nil {
		return err
	}
	log.Printf("ExpireData: %v(%v)", key, result)
	return nil
}

func (r *RedisCluster) ExpireGTData(key string, expiration time.Duration) error {
	ctx := context.Background()
	result, err := r.client.ExpireGT(ctx, key, expiration).Result()
	if err != nil {
		return err
	}
	log.Printf("ExpireGTData: %v(%v)", key, result)
	return nil
}

func (r *RedisCluster) ExistsKey(key string) (int64, error) {
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

func (r *RedisCluster) SAdd(key string, members ...interface{}) error {
	ctx := context.Background()
	count, err := r.client.SAdd(ctx, key, members).Result()
	if err != nil {
		return err
	}
	log.Printf("SAdd: %v(%v)", key, count)
	return nil
}

func (r *RedisCluster) SRem(key string, members ...interface{}) error {
	ctx := context.Background()
	count, err := r.client.SRem(ctx, key, members).Result()
	if err != nil {
		return err
	}
	log.Printf("SRem: %v(%v)", key, count)
	return nil
}

func (r *RedisCluster) SMIsMember(key string, members ...interface{}) (map[string]bool, error) {
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

func (r *RedisCluster) SMembers(key string) ([]string, error) {
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

func (r *RedisCluster) SCard(key string) (int64, error) {
	ctx := context.Background()
	val, err := r.client.SCard(ctx, key).Result()
	if err != nil {
		return val, err
	}
	log.Printf("SCard key: %v, val: %v\n", key, val)
	return val, nil
}
