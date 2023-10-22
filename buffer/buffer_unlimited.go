package buffer

import (
	"github.com/cornelk/hashmap"
)

var _ Buffer[string] = (*UnlimitedBuffer[string])(nil)

func NewUnlimitedBuffer[K BufferKey]() func(generator func(K) any) Buffer[K] {
	return func(generator func(K) any) Buffer[K] {
		return &UnlimitedBuffer[K]{
			generator: generator,
			items:     hashmap.New[K, any](),
		}
	}
}

type UnlimitedBuffer[K BufferKey] struct {
	generator func(K) any
	items     *hashmap.Map[K, any]
}

func (b UnlimitedBuffer[K]) GetOrInsert(key K) (any, bool) {
	return b.items.GetOrInsert(key, b.generator(key))
}
