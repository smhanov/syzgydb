package syzgydb

import (
	"container/list"
	"sync"
)

type cacheItem struct {
	key   string
	value []float64
}

type lruCache struct {
	mutex    sync.Mutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

func newLRUCache(capacity int) *lruCache {
	return &lruCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

func (c *lruCache) get(key string) ([]float64, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if element, found := c.items[key]; found {
		c.order.MoveToFront(element)
		return element.Value.(*cacheItem).value, true
	}
	return nil, false
}

func (c *lruCache) put(key string, value []float64) {
	if element, found := c.items[key]; found {
		c.order.MoveToFront(element)
		element.Value.(*cacheItem).value = value
		return
	}

	if c.order.Len() >= c.capacity {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.items, oldest.Value.(*cacheItem).key)
		}
	}

	item := &cacheItem{key: key, value: value}
	element := c.order.PushFront(item)
	c.items[key] = element
}
