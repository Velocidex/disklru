package disklru

import "sync"

type Counter struct {
	mu    sync.Mutex
	value int
}

func (self *Counter) Inc() {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.value++
}

func (self *Counter) Get() int {
	self.mu.Lock()
	defer self.mu.Unlock()

	return self.value
}

type Stats struct {
	Length, Size, Capacity, Evictions int64
	Hits, Misses                      int64
}

func (self *DiskLRU) Stats() Stats {
	return Stats{
		Size:   int64(self.opts.MaxSize),
		Hits:   int64(self.hit.Get()),
		Misses: int64(self.miss.Get()),
	}
}
