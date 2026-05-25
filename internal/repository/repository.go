package repository

import (
	"context"
	"time"

	"github.com/whenipush/envgate/internal/entity"
	"go.etcd.io/bbolt"
)

type repository struct {
	db *bbolt.DB
}

func NewRepository(db *bbolt.DB) *repository {
	return &repository{db: db}
}

const defaultTimeout = 10 * time.Second

func (r *repository) ListKeys(ctx context.Context, bucket entity.Bucket) ([][]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	var keys [][]byte

	err := r.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			key := make([]byte, len(k))
			copy(key, k)
			keys = append(keys, key)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (r *repository) Get(ctx context.Context, bucket entity.Bucket, key []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	var value []byte
	err := r.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}

		res := b.Get(key)
		if res == nil {
			return nil
		}

		value = make([]byte, len(res))
		copy(value, res)
		return nil
	})

	if err != nil {
		return nil, err
	}

	if value == nil {
		return nil, nil
	}

	return value, nil
}

func (r *repository) Put(ctx context.Context, bucket entity.Bucket, key []byte, value []byte) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	return r.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		return b.Put(key, value)
	})
}

func (r *repository) Delete(ctx context.Context, bucket entity.Bucket, key []byte) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		return b.Delete(key)
	})
}
func (r *repository) Scan(ctx context.Context, bucket entity.Bucket, cb func(k, v []byte) error) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return r.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			keyCopy := make([]byte, len(k))
			copy(keyCopy, k)

			valCopy := make([]byte, len(v))
			copy(valCopy, v)

			return cb(keyCopy, valCopy)
		})
	})
}
