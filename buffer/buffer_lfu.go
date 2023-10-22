package buffer

import (
	"github.com/bluele/gcache"
)

var _ Buffer[string] = (*LFUBuffer[string])(nil)

func NewLFUBuffer[K BufferKey](size int) func(generator func(K) any) Buffer[K] {
	return func(generator func(K) any) Buffer[K] {
		return &LFUBuffer[K]{
			generator: generator,
			items: gcache.New(size).
				LFU().
				LoaderFunc(func(k interface{}) (interface{}, error) {
					return generator(k.(K)), nil
				}).
				Build(),
		}
	}
}

type LFUBuffer[K BufferKey] struct {
	generator func(K) any
	items     gcache.Cache
}

func (b LFUBuffer[K]) GetOrInsert(key K) (any, bool) {
	item, err := b.items.Get(key)
	return item, err == nil
}
