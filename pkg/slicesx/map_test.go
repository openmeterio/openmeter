package slicesx

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestMap(t *testing.T) {
	s := []int{1, 2, 3}
	expected := []string{"1", "2", "3"}

	actual := Map(s, func(v int) string {
		return fmt.Sprintf("%d", v)
	})

	if !reflect.DeepEqual(expected, actual) {
		t.Fatal("map failed")
	}
}

func TestMapWithErrPreservingResults(t *testing.T) {
	firstErr := errors.New("first error")
	lastErr := errors.New("last error")

	// Given a mapper that returns usable results alongside errors.
	// When every input is mapped with its index.
	results, err := MapWithErrPreservingResults([]int{10, 20, 30}, func(value, index int) (int, error) {
		result := value + index
		switch index {
		case 0:
			return result, firstErr
		case 2:
			return result, lastErr
		default:
			return result, nil
		}
	})

	// Then all input-aligned results and both errors are returned.
	if !reflect.DeepEqual([]int{10, 21, 32}, results) {
		t.Fatalf("unexpected results: %v", results)
	}
	if !errors.Is(err, firstErr) || !errors.Is(err, lastErr) {
		t.Fatalf("expected joined errors, got %v", err)
	}
}
