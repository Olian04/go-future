package future

import (
	"context"
)

type State int

const (
	StatePending State = iota
	StateDone
	StateError
)

type Future[T any] struct {
	ctx     context.Context
	val     T
	err     error
	state   State
	stateCh chan State
}

func Ok[T any](ctx context.Context, val T) *Future[T] {
	return &Future[T]{
		ctx:   ctx,
		val:   val,
		state: StateDone,
	}
}

func Err[T any](ctx context.Context, err error) *Future[T] {
	return &Future[T]{
		ctx:   ctx,
		err:   err,
		state: StateError,
	}
}

func New[T any](ctx context.Context, fun func(ctx context.Context) (T, error)) *Future[T] {
	f := &Future[T]{
		ctx:     ctx,
		state:   StatePending,
		stateCh: make(chan State),
	}
	go func() {
		val, err := fun(f.ctx)
		if err != nil {
			f.err = err
			f.state = StateError
		} else {
			f.val = val
			f.state = StateDone
		}
		f.stateCh <- f.state
	}()
	return f
}

func (f *Future[T]) TryGet(ctx context.Context) (T, error) {
	if f.state == StateDone {
		return f.val, nil
	}
	if f.state == StateError {
		var defaultT T
		return defaultT, f.err
	}

	for {
		select {
		case state := <-f.stateCh:
			if state == StateDone {
				return f.val, nil
			}
			if state == StateError {
				var defaultT T
				return defaultT, f.err
			}
		case <-ctx.Done():
			var defaultT T
			return defaultT, ctx.Err()
		}
	}
}

func (f *Future[T]) GetOr(ctx context.Context, fallback T) T {
	v, err := f.TryGet(ctx)
	if err != nil {
		return fallback
	}
	return v
}

func (f *Future[T]) GetElse(ctx context.Context, fallback func() T) T {
	v, err := f.TryGet(ctx)
	if err != nil {
		return fallback()
	}
	return v
}

func (f *Future[T]) MustGet(ctx context.Context) T {
	v, err := f.TryGet(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

func Map[T any, U any](f *Future[T], fun func(ctx context.Context, val T) U) *Future[U] {
	f2 := New(f.ctx, func(ctx context.Context) (U, error) {
		val, err := f.TryGet(ctx)
		if err != nil {
			var defaultU U
			return defaultU, err
		}
		return fun(ctx, val), nil
	})
	return f2
}

func MapErr[T any](f *Future[T], fun func(ctx context.Context, val T) error) *Future[T] {
	f2 := New(f.ctx, func(ctx context.Context) (T, error) {
		val, err := f.TryGet(ctx)
		if err != nil {
			return val, fun(ctx, val)
		}
		return val, nil
	})
	return f2
}

func FlatMap[T any, U any](f *Future[T], fun func(ctx context.Context, val T) *Future[U]) *Future[U] {
	f2 := New(f.ctx, func(ctx context.Context) (U, error) {
		val, err := f.TryGet(ctx)
		if err != nil {
			var defaultU U
			return defaultU, err
		}
		return fun(ctx, val).TryGet(ctx)
	})
	return f2
}

func FlatMapErr[T any, U any](f *Future[T], fun func(ctx context.Context, val T) *Future[U]) *Future[U] {
	f2 := New(f.ctx, func(ctx context.Context) (U, error) {
		val, err := f.TryGet(ctx)
		if err != nil {
			return fun(ctx, val).TryGet(ctx)
		}
		var defaultU U
		return defaultU, nil
	})
	return f2
}

func IterPar[T any, U any](ctx context.Context, arr []T, fun func(ctx context.Context, val T) (U, error)) ([]U, error) {
	futures := make([]*Future[U], len(arr))
	for i, val := range arr {
		futures[i] = New(ctx, func(ctx context.Context) (U, error) {
			return fun(ctx, val)
		})
	}
	return All(ctx, futures)
}

func All[T any](ctx context.Context, futures []*Future[T]) ([]T, error) {
	doneCh := make(chan any)
	errCh := make(chan error)
	vals := make([]T, len(futures))

	for i, f := range futures {
		go func(f *Future[T], i int) {
			val, err := f.TryGet(ctx)
			if err != nil {
				errCh <- err
			} else {
				vals[i] = val
			}
			doneCh <- struct{}{}
		}(f, i)
	}

	for range futures {
		select {
		case <-doneCh:
		case err := <-errCh:
			return nil, err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return vals, nil
}
