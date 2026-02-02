package disklru

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/alecthomas/assert"
)

type MockClock struct {
	now int64
}

func (self *MockClock) Now() int64 {
	self.now++
	return self.now
}

func check(cache *DiskLRU, key int) error {
	key_str := fmt.Sprintf("%v", key)
	value, err := cache.Get(key_str)
	if err != nil {
		return err
	}

	if key_str != value.(string) {
		return fmt.Errorf("Unexpected value for key %v: %v", key, value)
	}

	return nil
}

func set(cache *DiskLRU, key int) error {
	key_str := fmt.Sprintf("%v", key)
	return cache.Set(key_str, key_str)
}

func TestLRU(t *testing.T) {
	fd, err := ioutil.TempFile("", "*.sqlite")
	assert.NoError(t, err)

	fd.Close()

	defer os.Remove(fd.Name())

	clock := &MockClock{
		now: 0,
	}

	opts := Options{
		Filename: fd.Name(),
		MaxSize:  3,
		Clock:    clock,

		// Manual housekeeping for tests.
		HouseKeepPeriodSec: -1,
		DEBUG:              true,
	}

	cache, err := NewDiskLRU(context.Background(), opts)
	assert.NoError(t, err)
	defer cache.Close()

	// Insert 5 items - overflowing the cache.
	for i := 0; i < 5; i++ {
		err := set(cache, i)
		assert.NoError(t, err)

		cache.Dump()

		err = check(cache, i)
		assert.NoError(t, err)
	}

	// All items are still in the cache
	assert.Equal(t, 5, len(cache.Items()))

	// run the housekeeping thread
	cache.HouseKeepOnce()

	// Older items are removed
	assert.Equal(t, 3, len(cache.Items()))

	// The most recent key is still there
	err = check(cache, 4)
	assert.NoError(t, err)

	// The first key is gone
	err = check(cache, 1)
	assert.Equal(t, err, KeyNotFoundError)

	// Advance the time a bit
	clock.now += 120 * NS

	// run the housekeeping thread
	cache.HouseKeepOnce()

	err = check(cache, 4)
	assert.Equal(t, err, KeyNotFoundError)

	// All items are removed due to ttl expiry
	assert.Equal(t, 0, len(cache.Items()))
}
