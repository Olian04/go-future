package future

import (
	"context"
)

type Future[T any] struct {
	fun   func(ctx context.Context) (T, error)
	ctx   context.Context
	valCh chan T
	errCh chan error
}

func Ok[T any](val T) *Future[T] {
	return New(context.Background(), func(ctx context.Context) (T, error) {
		return val, nil
	})
}

func Err[T any](err error) *Future[T] {
	return New(context.Background(), func(ctx context.Context) (T, error) {
		var defaultT T
		return defaultT, err
	})
}

func New[T any](ctx context.Context, fun func(ctx context.Context) (T, error)) *Future[T] {
	f := &Future[T]{
		fun:   fun,
		ctx:   ctx,
		valCh: make(chan T),
		errCh: make(chan error),
	}
	go func() {
		val, err := f.fun(f.ctx)
		if err != nil {
			f.errCh <- err
		} else {
			f.valCh <- val
		}
	}()
	return f
}

func (f *Future[T]) TryGet(ctx context.Context) (T, error) {
	select {
	case val := <-f.valCh:
		return val, nil
	case err := <-f.errCh:
		var defaultT T
		return defaultT, err
	case <-ctx.Done():
		var defaultT T
		return defaultT, ctx.Err()
	}
}

func GetOr[T any](f *Future[T], fallback func() T) T {
	v, err := f.TryGet(context.Background())
	if err != nil {
		return fallback()
	}
	return v
}

func MustGet[T any](f *Future[T]) T {
	v, err := f.TryGet(context.Background())
	if err != nil {
		panic(err)
	}
	return v
}

func Map[T any, U any](f *Future[T], fun func(ctx context.Context, val T) (U, error)) *Future[U] {
	f2 := New(f.ctx, func(ctx context.Context) (U, error) {
		val, err := f.TryGet(ctx)
		if err != nil {
			var defaultU U
			return defaultU, err
		}
		return fun(ctx, val)
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

func All[T any](ctx context.Context, futures []*Future[T]) ([]T, error) {
	type futureResult[T any] struct {
		Val   T
		Index int
	}
	innerCtx, cancel := context.WithCancel(ctx)
	valCh := make(chan futureResult[T], len(futures))
	errCh := make(chan error)

	for i, future := range futures {
		go func(f *Future[T], i int) {
			val, err := f.TryGet(innerCtx)
			if err != nil {
				errCh <- err
			} else {
				valCh <- futureResult[T]{Val: val, Index: i}
			}
		}(future, i)
	}

	vals := make([]T, 0, len(futures))
	for range len(futures) {
		select {
		case val := <-valCh:
			vals[val.Index] = val.Val
		case err := <-errCh:
			cancel()
			return nil, err
		case <-ctx.Done():
			cancel()
			return nil, ctx.Err()
		}
	}
	cancel()
	close(valCh)
	close(errCh)
	return vals, nil
}
