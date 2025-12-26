package rate

import (
	"context"
	"go.uber.org/ratelimit"
)

type Jitter struct {
	ch    chan struct{}
	l     ratelimit.Limiter
	limit int
}

func NewJitter(ctx context.Context, limit int) *Jitter {
	brst := int(float64(limit) * 0.1)
	if brst < 1 {
		brst = 1
	}
	jitter := &Jitter{
		limit: limit,
		ch:    make(chan struct{}, brst),
		l:     ratelimit.New(limit),
	}
	go jitter.provider(ctx)
	return jitter
}

func (l *Jitter) provider(ctx context.Context) {
	defer close(l.ch)
	for {
		l.l.Take()
		select {
		case <-ctx.Done():
			return
		case l.ch <- struct{}{}:
		}
	}
}

func (l *Jitter) Take() {
	<-l.ch
}

func (l *Jitter) Chan() <-chan struct{} {
	return l.ch
}
