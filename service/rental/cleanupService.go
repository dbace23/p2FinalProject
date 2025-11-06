package rental

import (
	rentalrepo "bookrental/repository/rental"
	"context"
	"time"
)

type Cleaner interface {
	ReleaseExpired(ctx context.Context) (int64, error)
}

type cleaner struct {
	r rentalrepo.Repo
}

func NewCleaner(r rentalrepo.Repo) Cleaner { return &cleaner{r: r} }

func (c *cleaner) ReleaseExpired(ctx context.Context) (int64, error) {
	return c.r.ReleaseExpiredBookings(ctx, time.Now().UTC())
}
