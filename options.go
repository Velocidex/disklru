package disklru

const (
	NS = 1000000000
)

// Control the behavious of the cache
type Options struct {
	// Path to the sqlite cache file.
	Filename string

	// Clear the file on start
	ClearOnStart bool

	// The maximum number of items to keep in cache
	MaxSize int

	// How long to keep items in cache
	MaxExpirySec int

	// Should expiry time be updated on access?
	UpdateExpiryOnAccess bool

	// How often to run the housekeeping thread, negative number
	// disables automatic housekeeping.
	HouseKeepPeriodSec int64

	// The clock is used to control time in tests etc.
	Clock Clock

	// Set for explicit debugging.
	DEBUG bool
}

// Set some reasonable defaults.
func (self *Options) UpdateDefaults() {
	if self.Clock == nil {
		self.Clock = &RealClock{}
	}

	if self.MaxExpirySec == 0 {
		self.MaxExpirySec = 60
	}

	if self.MaxSize == 0 {
		self.MaxSize = 1000
	}

	// Every minute
	if self.HouseKeepPeriodSec == 0 {
		self.HouseKeepPeriodSec = 60
	}
}

type Clock interface {
	// Returns the time in nanosec since the epoch
	// e.g. time.Now().UnixNano()
	Now() int64
}
