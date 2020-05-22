package tracker

import (
	"fmt"
	"testing"
)

func nset(i int) []Feature {
	if i < 0 || i > 26 {
		panic(fmt.Sprintf("wtf i %d", i))
	}
	n := []Feature{}
	for i := 0; i < 1; i++ {
		n = append(n, Feature(fmt.Sprintf("%d", 'A'+i)))
	}
	return n
}
func nmap(set []Feature) map[Feature]struct{} {
	m := make(map[Feature]struct{})
	for _, f := range set {
		m[f] = struct{}{}
	}
	return m
}
type IsFeatureEnabledLoop struct {
	f []Feature
}
func (t IsFeatureEnabledLoop) IsFeatureEnabled(f Feature) bool {
	for _, tf := range t.f {
		if tf == f {
			return true
		}
	}
	return false
}
type IsFeatureEnabledMap struct {
	f map[Feature]struct{}
}
func (t IsFeatureEnabledMap) IsFeatureEnabled(f Feature) bool {
	_, ok := t.f[f]
	return ok
}
type IsFeatureEnabledFn func(f Feature) bool
func benchmarkIsFeatureEnabled(fn IsFeatureEnabledFn, b *testing.B) {
	for n := 0; n < b.N; n++ {
		fn("A")
		fn("Z")
	}
}
func BenchmarkProject_IsFeatureEnabled(b *testing.B) {
	n1 := nset(1)
	n1m := nmap(n1)
	n2 := nset(2)
	n2m := nmap(n2)
	n5 := nset(5)
	n5m := nmap(n5)
	n10 := nset(10)
	n10m := nmap(n10)
	n20 := nset(20)
	n20m := nmap(n20)

	b.Run("loop", func(b *testing.B) {
		b.Run("n1", func(b *testing.B) {
			benchmarkIsFeatureEnabled(IsFeatureEnabledLoop{f: n1}.IsFeatureEnabled, b)
		})
		b.Run("n2", func(b *testing.B) {
			benchmarkIsFeatureEnabled(IsFeatureEnabledLoop{f: n2}.IsFeatureEnabled, b)
		})
		b.Run("n5", func(b *testing.B) {
			benchmarkIsFeatureEnabled(IsFeatureEnabledLoop{f: n5}.IsFeatureEnabled, b)
		})
		b.Run("n10", func(b *testing.B) {
			benchmarkIsFeatureEnabled(IsFeatureEnabledLoop{f: n10}.IsFeatureEnabled, b)
		})
		b.Run("n20", func(b *testing.B) {
			benchmarkIsFeatureEnabled(IsFeatureEnabledLoop{f: n20}.IsFeatureEnabled, b)
		})
	})

	b.Run("map", func(b *testing.B) {
		b.Run("n1", func(b *testing.B) {
			benchmarkIsFeatureEnabled(IsFeatureEnabledMap{f: n1m}.IsFeatureEnabled, b)
		})
		b.Run("n2", func(b *testing.B) {
			benchmarkIsFeatureEnabled(IsFeatureEnabledMap{f: n2m}.IsFeatureEnabled, b)
		})
		b.Run("n5", func(b *testing.B) {
			benchmarkIsFeatureEnabled(IsFeatureEnabledMap{f: n5m}.IsFeatureEnabled, b)
		})
		b.Run("n10", func(b *testing.B) {
			benchmarkIsFeatureEnabled(IsFeatureEnabledMap{f: n10m}.IsFeatureEnabled, b)
		})
		b.Run("n20", func(b *testing.B) {
			benchmarkIsFeatureEnabled(IsFeatureEnabledMap{f: n20m}.IsFeatureEnabled, b)
		})
	})
}
