package buffer

type BufferKey interface {
	~string | ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

type Buffer[K BufferKey] interface {
	GetOrInsert(K) (any, bool)
}
