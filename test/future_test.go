package test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Olian04/go-future/future"
)

func TestFuture(t *testing.T) {
	ctx := context.Background()
	f := future.New(ctx, func(ctx context.Context) (int, error) {
		return 1, nil
	})

	val, err := f.TryGet(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if val != 1 {
		t.Fatalf("expected 1, got %v", val)
	}
}

func TestFutureError(t *testing.T) {
	ctx := context.Background()
	f := future.New(ctx, func(ctx context.Context) (int, error) {
		return 0, errors.New("error")
	})

	_, err := f.TryGet(ctx)
	if err == nil || err.Error() != "error" {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestGetOr(t *testing.T) {
	ctx := context.Background()
	f := future.New(ctx, func(ctx context.Context) (int, error) {
		return 0, errors.New("error")
	})

	val := f.GetOr(ctx, 1)
	if val != 1 {
		t.Fatalf("expected 1, got %v", val)
	}
}

func TestGetElse(t *testing.T) {
	ctx := context.Background()
	f := future.New(ctx, func(ctx context.Context) (int, error) {
		return 0, errors.New("error")
	})

	val := f.GetElse(ctx, func() int {
		return 1
	})
	if val != 1 {
		t.Fatalf("expected 1, got %v", val)
	}
}

func TestMustGet(t *testing.T) {
	ctx := context.Background()
	f := future.New(ctx, func(ctx context.Context) (int, error) {
		return 0, errors.New("error")
	})
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()
	f.MustGet(ctx)
}

func TestMap(t *testing.T) {
	ctx := context.Background()
	f := future.New(ctx, func(ctx context.Context) (int, error) {
		return 1, nil
	})

	mapped := future.Map(f, func(ctx context.Context, val int) string {
		return fmt.Sprintf("%d", val)
	})

	val, err := mapped.TryGet(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if val != "1" {
		t.Fatalf("expected 1, got %v", val)
	}
}

func TestMapError(t *testing.T) {
	ctx := context.Background()
	f := future.New(ctx, func(ctx context.Context) (int, error) {
		return 1, errors.New("error")
	})

	mapped := future.MapErr(f, func(ctx context.Context, val int) error {
		return errors.New("mapped error")
	})

	_, err := mapped.TryGet(ctx)
	if err == nil || err.Error() != "mapped error" {
		t.Fatalf("expected mapped error, got %v", err)
	}
}

func TestFlatMap(t *testing.T) {
	ctx := context.Background()
	f := future.New(ctx, func(ctx context.Context) (int, error) {
		return 1, nil
	})

	flatMapped := future.FlatMap(f, func(ctx context.Context, val int) *future.Future[string] {
		return future.Ok(ctx, fmt.Sprintf("%d", val))
	})

	val, err := flatMapped.TryGet(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if val != "1" {
		t.Fatalf("expected 1, got %v", val)
	}
}

func TestAll(t *testing.T) {
	ctx := context.Background()
	f1 := future.New(ctx, func(ctx context.Context) (int, error) {
		return 1, nil
	})
	f2 := future.New(ctx, func(ctx context.Context) (int, error) {
		return 2, nil
	})

	all, err := future.All(ctx, []*future.Future[int]{f1, f2})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 values, got %v", len(all))
	}
	if all[0] != 1 {
		t.Fatalf("expected 1, got %v", all[0])
	}
	if all[1] != 2 {
		t.Fatalf("expected 2, got %v", all[1])
	}
}

func TestIterPar(t *testing.T) {
	ctx := context.Background()
	arr := []int{1, 2, 3, 4, 5}
	vals, err := future.IterPar(ctx, arr, func(ctx context.Context, val int) (int, error) {
		return val * 2, nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(vals) != 5 {
		t.Fatalf("expected 5 values, got %v", len(vals))
	}
	for i, val := range vals {
		if val != arr[i]*2 {
			t.Fatalf("expected %d, got %v", arr[i]*2, val)
		}
	}
}

func Fibbonaci(n int) int {
	if n <= 1 {
		return n
	}
	return Fibbonaci(n-1) + Fibbonaci(n-2)
}

func BenchmarkIterBaseline(b *testing.B) {
	arr := make([]int, 100_000)
	for b.Loop() {
		for range arr {
			Fibbonaci(15)
		}
	}
}

func BenchmarkIterPar(b *testing.B) {
	ctx := context.Background()
	arr := make([]int, 100_000)
	for b.Loop() {
		future.IterPar(ctx, arr, func(ctx context.Context, val int) (int, error) {
			return Fibbonaci(15), nil
		})
	}
}
