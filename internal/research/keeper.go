package research

import (
	"errors"
	"sync"
	"time"

	"newsbot/internal/schedule"
	"newsbot/internal/watchlist"
)

var (
	ErrNewsEmpty = errors.New("новина без монети або тексту")
	ErrNoNews    = errors.New("такої новини немає")
	ErrPlanEmpty = errors.New("план порожній")
	ErrNoPlan    = errors.New("плану ще немає")
)

type Store interface {
	Save(archive Archive) error
}

type Keeper struct {
	store   Store
	mu      sync.Mutex
	archive Archive
}

func NewKeeper(store Store, archive Archive) *Keeper {
	return &Keeper{store: store, archive: archive}
}

func (b *Keeper) Of(asset string) []Finding {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.archive.Of(watchlist.Normalize(asset))
}

func (b *Keeper) Dates(zone *time.Location) []schedule.Entry {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.archive.Dates(zone)
}

func (b *Keeper) Keep(f Finding) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	archive, kept := b.archive.Keep(f)
	if !kept {
		return ErrNewsEmpty
	}

	return b.keep(archive)
}

func (b *Keeper) Remove(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	archive, removed := b.archive.Remove(id)
	if !removed {
		return ErrNoNews
	}

	return b.keep(archive)
}

func (b *Keeper) Plan(asset string) (Plan, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.archive.Plan(watchlist.Normalize(asset))
}

func (b *Keeper) FixPlan(asset string, plan Plan) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	archive, fixed := b.archive.Fix(watchlist.Normalize(asset), plan)
	if !fixed {
		return ErrPlanEmpty
	}

	return b.keep(archive)
}

func (b *Keeper) keep(archive Archive) error {
	if err := b.store.Save(archive); err != nil {
		return err
	}

	b.archive = archive

	return nil
}
