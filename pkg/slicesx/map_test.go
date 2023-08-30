package slicesx

import (
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
