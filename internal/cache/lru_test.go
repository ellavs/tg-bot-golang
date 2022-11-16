package cache

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLRU_Add_existElementWithFullQueueSync_moveToFront(t *testing.T) {
	//Arrange
	lru := NewLRU(3)
	lru.Add("someKey1", 8)
	lru.Add("someKey2", "56")
	emptyMap := make(map[string]int)
	lru.Add("someKey3", emptyMap)

	//Act
	lru.Add("someKey1", 10)

	//Assert
	frontItem := lru.queue.Front().Value.(*Item)
	backItem := lru.queue.Back().Value.(*Item)
	assert.Equal(t, "someKey1", frontItem.Key)
	assert.Equal(t, 10, frontItem.Value)
	assert.Equal(t, "someKey2", backItem.Key)
	assert.Equal(t, "56", backItem.Value)
	assert.Equal(t, 3, lru.queue.Len())
}

func TestLRU_Add_existElementSync_moveToFront(t *testing.T) {
	//Arrange
	lru := NewLRU(3)
	lru.Add("someKey1", 8)
	lru.Add("someKey2", "56")

	//Act
	lru.Add("someKey1", 10)

	//Assert
	frontItem := lru.queue.Front().Value.(*Item)
	backItem := lru.queue.Back().Value.(*Item)
	assert.Equal(t, "someKey1", frontItem.Key)
	assert.Equal(t, 10, frontItem.Value)
	assert.Equal(t, "someKey2", backItem.Key)
	assert.Equal(t, "56", backItem.Value)
	assert.Equal(t, 2, lru.queue.Len())
}

func TestLRU_Add_newElementWithFullQueueSync_clearAndPushToFront(t *testing.T) {
	//Arrange
	lru := NewLRU(3)
	lru.Add("someKey1", 8)
	lru.Add("someKey2", "56")
	lru.Add("someKey3", Item{"key", 7})

	//Act
	lru.Add("someKey4", 5)

	//Assert
	frontItem := lru.queue.Front().Value.(*Item)
	backItem := lru.queue.Back().Value.(*Item)
	assert.Equal(t, "someKey4", frontItem.Key)
	assert.Equal(t, 5, frontItem.Value)
	assert.Equal(t, "someKey2", backItem.Key)
	assert.Equal(t, "56", backItem.Value)
	assert.Equal(t, 3, lru.queue.Len())
	assert.Nil(t, lru.Get("someKey1"))
}

func TestLRU_Add_newElementSync_pushToFront(t *testing.T) {
	//Arrange
	lru := NewLRU(3)
	lru.Add("someKey1", 8)

	//Act
	lru.Add("someKey2", 5)

	//Assert
	frontItem := lru.queue.Front().Value.(*Item)
	backItem := lru.queue.Back().Value.(*Item)
	assert.Equal(t, "someKey2", frontItem.Key)
	assert.Equal(t, 5, frontItem.Value)
	assert.Equal(t, "someKey1", backItem.Key)
	assert.Equal(t, 8, backItem.Value)
	assert.Equal(t, 2, lru.queue.Len())
}

func TestLRU_Add_newElementAsync_allKeysExists(t *testing.T) {
	//Arrange
	wg := sync.WaitGroup{}
	lru := NewLRU(3)
	wg.Add(3)

	//Act
	go func() {
		lru.Add("someKey1", 8)
		wg.Done()
	}()
	go func() {
		lru.Add("someKey2", "56")
		wg.Done()
	}()
	go func() {
		lru.Add("someKey3", Item{"key", 7})
		wg.Done()
	}()

	wg.Wait()

	//Assert
	assert.Equal(t, 8, lru.Get("someKey1"))
	assert.Equal(t, "56", lru.Get("someKey2"))
	assert.Equal(t, Item{"key", 7}, lru.Get("someKey3"))
	assert.Equal(t, 3, lru.queue.Len())
}

func TestLRU_Get_hasElement_returnItAndMoveToFront(t *testing.T) {
	//Arrange
	lru := NewLRU(3)
	lru.Add("someKey1", 8)
	lru.Add("someKey2", 5)
	lru.Add("someKey3", "90")

	//Act
	item := lru.Get("someKey2")

	//Assert
	frontItem := lru.queue.Front().Value.(*Item)
	backItem := lru.queue.Back().Value.(*Item)
	assert.Equal(t, 5, item)
	assert.Equal(t, "someKey2", frontItem.Key)
	assert.Equal(t, 5, frontItem.Value)
	assert.Equal(t, "someKey1", backItem.Key)
	assert.Equal(t, 8, backItem.Value)
	assert.Equal(t, 3, lru.queue.Len())
}

func TestLRU_Get_hasNotElement_returnNil(t *testing.T) {
	//Arrange
	lru := NewLRU(3)
	lru.Add("someKey1", 8)
	lru.Add("someKey2", 5)
	lru.Add("someKey3", "90")

	//Act
	item := lru.Get("someKey")

	//Assert
	frontItem := lru.queue.Front().Value.(*Item)
	backItem := lru.queue.Back().Value.(*Item)
	assert.Nil(t, item)
	assert.Equal(t, "someKey3", frontItem.Key)
	assert.Equal(t, "90", frontItem.Value)
	assert.Equal(t, "someKey1", backItem.Key)
	assert.Equal(t, 8, backItem.Value)
	assert.Equal(t, 3, lru.queue.Len())
}

func TestLRU_Remove_hasElement_removeIt(t *testing.T) {
	//Arrange
	lru := NewLRU(3)
	lru.Add("someKey1", 8)
	lru.Add("someKey2", 5)
	lru.Add("someKey3", "90")

	//Act
	lru.Remove("someKey2")

	//Assert
	frontItem := lru.queue.Front().Value.(*Item)
	backItem := lru.queue.Back().Value.(*Item)
	assert.Nil(t, lru.Get("someKey2"))
	assert.Equal(t, "someKey3", frontItem.Key)
	assert.Equal(t, "90", frontItem.Value)
	assert.Equal(t, "someKey1", backItem.Key)
	assert.Equal(t, 8, backItem.Value)
	assert.Equal(t, 2, lru.queue.Len())
}

func TestLRU_Remove_hasNotElement_doNothing(t *testing.T) {
	//Arrange
	lru := NewLRU(3)
	lru.Add("someKey1", 8)
	lru.Add("someKey2", 5)
	lru.Add("someKey3", "90")

	//Act
	lru.Remove("someKey")

	//Assert
	frontItem := lru.queue.Front().Value.(*Item)
	backItem := lru.queue.Back().Value.(*Item)
	assert.Equal(t, "someKey3", frontItem.Key)
	assert.Equal(t, "90", frontItem.Value)
	assert.Equal(t, "someKey1", backItem.Key)
	assert.Equal(t, 8, backItem.Value)
	assert.Equal(t, 3, lru.queue.Len())
}
