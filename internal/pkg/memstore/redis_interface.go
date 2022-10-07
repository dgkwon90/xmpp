package memstore

import (
	"time"
)

type Redis interface {
	Close() error
	Connect() error

	// Basic
	SetData(key string, val interface{}, duration time.Duration) error
	GetData(key string) (string, error)
	DelData(key string) error
	ExpireData(key string, expiration time.Duration) error
	ExpireGTData(key string, expiration time.Duration) error
	ExistsKey(key string) (int64, error)

	// Sets
	SAdd(key string, members ...interface{}) error
	SRem(key string, members ...interface{}) error
	SMIsMember(key string, members ...interface{}) (map[string]bool, error)
	SMembers(key string) ([]string, error)
	SCard(key string) (int64, error)
}
