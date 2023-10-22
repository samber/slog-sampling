package buffer

import (
	"github.com/bluele/gcache"
)

var _ Buffer[string] = (*LRUBuffer[string])(nil)

func NewLRUBuffer[K BufferKey](size int) func(generator func(K) any) Buffer[K] {
	return func(generator func(K) any) Buffer[K] {
		return &LRUBuffer[K]{
			generator: generator,
			items: gcache.New(size).
				LRU().
				LoaderFunc(func(k interface{}) (interface{}, error) {
					return generator(k.(K)), nil
				}).
				Build(),
		}
	}
}

type LRUBuffer[K BufferKey] struct {
	generator func(K) any
	items     gcache.Cache
}

func (b LRUBuffer[K]) GetOrInsert(key K) (any, bool) {
	item, err := b.items.Get(key)
	return item, err == nil
}
