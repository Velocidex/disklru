package disklru

import (
	"fmt"
)

func (self *DiskLRU) Debug(msg string, args ...interface{}) {
	if self.opts.DEBUG {
		fmt.Printf("DiskLRU DEBUG "+self.opts.Filename+": "+msg+"\n", args...)
	}
}

func (self *DiskLRU) Dump() {
	if !self.opts.DEBUG {
		return
	}

	rows, err := self.handle.Query("SELECT key, expires, lastAccess FROM cache")
	if err != nil {
		self.Debug("Dump: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var expires, lastAccess int
		err := rows.Scan(&key, &expires, &lastAccess)
		if err == nil {
			self.Debug("Dump: Key %v, expires %v, lastAccess %v",
				key, expires, lastAccess)
		}
	}

}
