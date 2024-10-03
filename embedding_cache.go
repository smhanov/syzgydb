package syzgydb

import (
	"container/list"
)

const maxCacheSize = 100

type cacheItem struct {
	key   string
	value []float64
}

type LRUCache struct {
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

func (c *LRUCache) Get(key string) ([]float64, bool) {
	if element, found := c.items[key]; found {
		c.order.MoveToFront(element)
		return element.Value.(*cacheItem).value, true
	}
	return nil, false
}

func (c *LRUCache) Put(key string, value []float64) {
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
