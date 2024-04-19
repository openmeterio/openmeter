package inmemory_lock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"

	credit_model "github.com/openmeterio/openmeter/pkg/credit"
)

func NewLockManager(duration time.Duration) credit_model.LockManager {
	return &LockManager{
		duration: duration,
		locks:    map[string]Lock{},
	}
}

type LockManager struct {
	duration time.Duration
	mu       sync.Mutex
	locks    map[string]Lock
}

func (lm *LockManager) Obtain(ctx context.Context, namespace string, subject string) (credit_model.Lock, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lock := newLock(namespace, subject, lm.duration)
	key := lock.Key()

	// Check if the lock is already held
	existingLock, ok := lm.locks[key]
	if ok {
		if existingLock.isExpired() {
			delete(lm.locks, key)
		} else {
			return &Lock{}, &credit_model.LockErrNotObtained{
				Namespace: namespace,
				Subject:   subject,
			}
		}
	}

	lm.locks[key] = lock
	return &lock, nil
}

func (lm *LockManager) Release(ctx context.Context, lock credit_model.Lock) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	key := lock.Key()
	delete(lm.locks, key)

	return nil
}

func (lm *LockManager) Extend(ctx context.Context, lock credit_model.Lock) error {
	key := lock.Key()

	if _, ok := lm.locks[key]; !ok {
		return &credit_model.LockNotHeld{
			Namespace: lock.Namespace(),
			Subject:   lock.Subject(),
		}
	}

	// Create a new lock with the same ID
	nl := newLock(lock.Namespace(), lock.Subject(), lm.duration)
	nl.id = lock.ID()

	lm.locks[key] = nl

	return nil
}

func newLock(namespace, subject string, duration time.Duration) Lock {
	return Lock{
		id:        ulid.Make().String(),
		namespace: namespace,
		subject:   subject,
		expiresAt: time.Now().Add(duration),
	}
}

type Lock struct {
	id        string
	namespace string
	subject   string
	expiresAt time.Time
}

func (l *Lock) ID() string {
	return l.id
}

func (l *Lock) Key() string {
	return fmt.Sprintf("%s:%s", l.namespace, l.subject)
}

func (l *Lock) Namespace() string {
	return l.namespace
}

func (l *Lock) Subject() string {
	return l.subject
}

func (l *Lock) isExpired() bool {
	return time.Now().After(l.expiresAt)
}
