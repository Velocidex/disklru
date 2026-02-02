package disklru

import (
	"database/sql"
	"time"
)

func _now(self *DiskLRU) sql.NamedArg {
	return sql.Named("now", self.opts.Clock.Now())
}

func _key(key string) sql.NamedArg {
	return sql.Named("key", key)
}

func _value(value []byte) sql.NamedArg {
	return sql.Named("value", value)
}

func _expires(self *DiskLRU) sql.NamedArg {
	return sql.Named("expires", self.opts.Clock.Now()+
		NS*int64(self.opts.MaxExpirySec))
}

func _max_items(self *DiskLRU) sql.NamedArg {
	return sql.Named("maxItems", self.opts.MaxSize)
}

type RealClock struct{}

func (self *RealClock) Now() int64 {
	ts := time.Now().UnixNano()
	return ts
}
