// Based on https://github.com/jkelin/cache-sqlite-lru-ttl/

package disklru

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Encoder interface {
	Encode(obj interface{}) ([]byte, error)
	Decode(in []byte) (interface{}, error)
}

type DiskLRU struct {
	handle  *sql.DB
	encoder Encoder

	set_stm, get_stm, get_update_expiry_stm,
	peek_stm, delete_stm,
	clear_stm, cleanup_expires_stm,
	cleanup_lru_stm *sql.Stmt

	opts Options

	hit, miss Counter

	cancel func()
}

func (self *DiskLRU) HouseKeepOnce() {
	self.Debug("Housekeeping run")

	self.cleanup_lru_stm.Exec(_max_items(self))
	self.cleanup_expires_stm.Exec(_now(self))
}

func (self *DiskLRU) houseKeeping(ctx context.Context) {
	if self.opts.HouseKeepPeriodSec < 0 {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-time.After(time.Second *
			time.Duration(self.opts.HouseKeepPeriodSec)):
			self.HouseKeepOnce()
		}
	}
}

func (self *DiskLRU) Close() error {
	self.Debug("Close")
	self.cancel()
	self.set_stm.Close()
	self.get_stm.Close()
	self.get_update_expiry_stm.Close()
	self.peek_stm.Close()
	self.delete_stm.Close()
	self.clear_stm.Close()
	self.cleanup_lru_stm.Close()
	self.cleanup_expires_stm.Close()

	return self.handle.Close()
}

func (self *DiskLRU) initTables() error {
	_, err := self.handle.Exec(`
 PRAGMA journal_mode=WAL;
 PRAGMA busy_timeout=60000;
 PRAGMA synchronous=NORMAL;

 CREATE TABLE IF NOT EXISTS cache (
   key TEXT PRIMARY KEY,
   value BLOB,
   expires BIGINT,
   lastAccess BIGINT
 );

 CREATE UNIQUE INDEX IF NOT EXISTS key ON cache (key);
 CREATE INDEX IF NOT EXISTS expires ON cache (expires);
 CREATE INDEX IF NOT EXISTS lastAccess ON cache (lastAccess);

`)
	return err
}

func (self *DiskLRU) Set(key string, value interface{}) error {
	buf, err := self.encoder.Encode(value)
	if err != nil {
		return err
	}

	_, err = self.set_stm.Exec(
		_now(self), _key(key), _value(buf), _expires(self))
	self.Debug("key %v set %v error: %v", key, string(buf), err)

	return err
}

func (self *DiskLRU) Get(key string) (interface{}, error) {
	var buf []byte

	if self.opts.UpdateExpiryOnAccess {
		err := self.get_update_expiry_stm.QueryRow(
			_now(self), _key(key), _expires(self)).Scan(&buf)
		if err != nil {
			self.miss.Inc()
			self.Debug("Get Key not found %v", key)
			return nil, KeyNotFoundError
		}

	} else {
		err := self.get_stm.QueryRow(_now(self), _key(key)).Scan(&buf)
		if err != nil {
			self.miss.Inc()
			self.Debug("Get Key not found %v: %v", key, err)
			return nil, KeyNotFoundError
		}
	}

	self.hit.Inc()
	self.Debug("Get Key %v: %v", key, string(buf))
	return self.encoder.Decode(buf)
}

type CacheItem struct {
	Key   string
	Value interface{}
}

func (self *DiskLRU) Items() (res []CacheItem) {
	now := time.Now()

	rows, err := self.handle.Query("SELECT key, value FROM cache")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var buf []byte
		err := rows.Scan(&key, &buf)
		if err != nil {
			continue
		}

		value, err := self.encoder.Decode(buf)
		if err != nil {
			continue
		}
		res = append(res, CacheItem{
			Key:   key,
			Value: value,
		})
	}

	self.Debug("Items call returned %v items in %v", len(res),
		time.Now().Sub(now).Round(time.Millisecond).String())

	return res
}

func (self *DiskLRU) Peek(key string) (interface{}, error) {
	var buf []byte

	err := self.peek_stm.QueryRow(_key(key)).Scan(&buf)
	if err != nil {
		return nil, err
	}

	return self.encoder.Decode(buf)
}

func (self *DiskLRU) SetEncoder(encoder Encoder) {
	self.encoder = encoder
}

func (self *DiskLRU) Delete(key string) bool {
	self.Debug("Delete key %v", key)
	self.delete_stm.Exec(_key(key))
	return false
}

func NewDiskLRU(
	ctx context.Context, opts Options) (*DiskLRU, error) {

	opts.UpdateDefaults()

	handle, err := sql.Open("sqlite3", opts.Filename)
	if err != nil {
		return nil, err
	}

	sub_ctx, cancel := context.WithCancel(ctx)

	self := &DiskLRU{
		handle:  handle,
		encoder: JsonEncoder{},
		opts:    opts,
		cancel:  cancel,
	}

	err = self.initTables()
	if err != nil {
		return nil, err
	}

	self.set_stm, err = handle.Prepare(
		`REPLACE INTO cache(key, value, expires, lastAccess)
        VALUES (@key, @value, @expires, @now)`)
	if err != nil {
		return nil, err
	}

	self.get_stm, err = handle.Prepare(
		`UPDATE OR IGNORE cache
         SET lastAccess = @now
         WHERE key = @key AND (expires > @now OR expires IS NULL)
         RETURNING value`)
	if err != nil {
		return nil, err
	}

	self.get_update_expiry_stm, err = handle.Prepare(
		`UPDATE OR IGNORE cache
         SET lastAccess = @now, expires = @expires
         WHERE key = @key AND (expires > @now OR expires IS NULL)
         RETURNING value`)
	if err != nil {
		return nil, err
	}

	self.peek_stm, err = handle.Prepare(
		`SELECT value FROM cache
         WHERE key = @key`)
	if err != nil {
		return nil, err
	}

	self.delete_stm, err = handle.Prepare(
		`DELETE FROM cache WHERE key = @key`)
	if err != nil {
		return nil, err
	}

	self.clear_stm, err = handle.Prepare(`DELETE FROM cache`)
	if err != nil {
		return nil, err
	}

	self.cleanup_expires_stm, err = handle.Prepare(
		`DELETE FROM cache WHERE expires < @now`)
	if err != nil {
		return nil, err
	}

	self.cleanup_lru_stm, err = handle.Prepare(
		`DELETE FROM cache
         WHERE key IN (
           SELECT key FROM cache
           ORDER BY lastAccess ASC
        LIMIT MAX(0, (SELECT COUNT(*) - @maxItems FROM cache)))`)
	if err != nil {
		return nil, err
	}

	if opts.ClearOnStart {
		self.clear_stm.Exec()
	}

	// Run the housekeep thread
	go self.houseKeeping(sub_ctx)

	return self, nil
}
