package slogsampling

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomPercentage(t *testing.T) {
	is := assert.New(t)

	for i := 0; i < 10000; i++ {
		r := randomPercentage()
		is.True(r >= 0 && r < 1)
	}
}
