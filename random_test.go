package slogsampling

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomPercentage(t *testing.T) {
	is := assert.New(t)

	for i := 1; i < 10000; i++ {
		r, err := randomPercentage(int64(i))
		is.NoError(err)
		is.True(r >= 0 && r < 1)
	}
}
