// Package cache - пакет для работы с LRU кэшем
package cache

import (
	"container/list"
	"sync"
)

type Item struct {
	Key   string
	Value any
}

type LRU struct {
	mutex    *sync.RWMutex
	capacity int
	queue    *list.List
	items    map[string]*list.Element
}

func NewLRU(capacity int) *LRU {
	return &LRU{
		mutex:    new(sync.RWMutex),
		capacity: capacity,
		queue:    list.New(),
		items:    make(map[string]*list.Element),
	}
}

// Add сохранить значение в кэш по заданному ключу.
func (c *LRU) Add(key string, value any) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if element, exists := c.items[key]; exists {
		c.queue.MoveToFront(element)
		element.Value.(*Item).Value = value
		return
	}

	if c.queue.Len() == c.capacity {
		c.clear()
	}

	item := &Item{
		Key:   key,
		Value: value,
	}

	element := c.queue.PushFront(item)
	c.items[item.Key] = element
}

// Get получить значение из кэша по заданному ключу.
func (c *LRU) Get(key string) any {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	element, exists := c.items[key]
	if !exists {
		return nil
	}

	c.queue.MoveToFront(element)
	return element.Value.(*Item).Value
}

func (c *LRU) Remove(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if val, found := c.items[key]; found {
		c.deleteItem(val)
	}
}

func (c *LRU) Len() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.items)
}

func (c *LRU) clear() {
	if element := c.queue.Back(); element != nil {
		c.deleteItem(element)
	}
}

func (c *LRU) deleteItem(element *list.Element) {
	item := c.queue.Remove(element).(*Item)
	delete(c.items, item.Key)
}
