package buffer

import (
	"github.com/bluele/gcache"
)

var _ Buffer[string] = (*ARCBuffer[string])(nil)

func NewARCBuffer[K BufferKey](size int) func(generator func(K) any) Buffer[K] {
	return func(generator func(K) any) Buffer[K] {
		return &ARCBuffer[K]{
			generator: generator,
			items: gcache.New(size).
				ARC().
				LoaderFunc(func(k interface{}) (interface{}, error) {
					return generator(k.(K)), nil
				}).
				Build(),
		}
	}
}

type ARCBuffer[K BufferKey] struct {
	generator func(K) any
	items     gcache.Cache
}

func (b ARCBuffer[K]) GetOrInsert(key K) (any, bool) {
	item, err := b.items.Get(key)
	return item, err == nil
}
