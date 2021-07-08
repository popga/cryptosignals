package cache

import (
	"sync"
)

type Cache struct {
	Data map[string]interface{}
	sync.Mutex
}

func New() *Cache {
	return &Cache{
		Data:  make(map[string]interface{}),
		Mutex: sync.Mutex{},
	}
}
func (c *Cache) Set(key string, value interface{}) {
	c.Lock()
	c.Data[key] = value
	c.Unlock()
}
func (c *Cache) Get(key string) interface{} {
	c.Lock()
	v, ok := c.Data[key]
	c.Unlock()
	if ok {
		return v
	}
	return nil
	//return c.Data[key]
	//return nil
}
func (c *Cache) Exists(key string) bool {
	c.Lock()
	_, ok := c.Data[key]
	c.Unlock()
	if ok {
		return true
	}
	return false
}
func (c *Cache) Delete(key string) {
	c.Lock()
	_, ok := c.Data[key]
	c.Unlock()
	if ok {
		c.Lock()
		delete(c.Data, key)
		c.Unlock()
	}
}
