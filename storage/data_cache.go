package storage

import (
	"time"

	cache "github.com/patrickmn/go-cache"
)

type Cache interface {
	Get(string) ([]byte, bool)
	Set(dbOperation, string, []byte)
	Clear()
}

type dbOperation int

const (
	dbPut = iota
	dbDelete
)

const (
	defaultTimeout    = 1 * time.Minute
	defaultExpiration = 2 * time.Minute
)

type dbCache struct {
	cache *cache.Cache
}

type cacheData struct {
	op    dbOperation
	value []byte
}

func newCache() *dbCache {
	return &dbCache{
		cache: cache.New(defaultTimeout, defaultExpiration),
	}
}

func (c *dbCache) Get(key string) ([]byte, bool) {
	obj, found := c.cache.Get(key)
	if !found {
		return []byte{}, found
	}

	data := obj.(cacheData)
	// if key is deleted, then cache should return not found
	if dbDelete == data.op {
		return []byte{}, false
	}

	return data.value, found
}

func (c *dbCache) Set(op dbOperation, key string, value []byte) {
	cached := cacheData{
		op:    op,
		value: value,
	}
	c.cache.Set(key, cached, defaultExpiration)
}

func (c *dbCache) Clear() {
	c.cache.Flush()
}
