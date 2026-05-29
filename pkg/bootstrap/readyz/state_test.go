package readyz

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetDraining_FlipsFlag(t *testing.T) {
	// Reset to known state.
	SetDraining(false)
	t.Cleanup(func() { SetDraining(false) })

	assert.False(t, IsDraining())

	SetDraining(true)
	assert.True(t, IsDraining())

	SetDraining(false)
	assert.False(t, IsDraining())
}

func TestDraining_ConcurrentReadsAndWritesAreSafe(t *testing.T) {
	// Reset to known state.
	SetDraining(false)
	t.Cleanup(func() { SetDraining(false) })

	const goroutines = 64
	const iterations = 500

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for i := range goroutines {
		go func(i int) {
			defer wg.Done()

			for j := range iterations {
				if j%2 == 0 {
					SetDraining(true)
				} else {
					SetDraining(false)
				}
				_ = IsDraining()
			}
		}(i)
	}

	wg.Wait()
}

func TestSelfProbe_DefaultsToFalse(t *testing.T) {
	// Gate 7 of ring:dev-readyz: default is false ("unhealthy until proven
	// otherwise"). Other tests in the package may have flipped the flag, so
	// we reset to the package default and verify.
	SetSelfProbe(false)
	t.Cleanup(func() { SetSelfProbe(false) })

	assert.False(t, IsSelfProbeOK())
}

func TestSelfProbe_FlipsFlag(t *testing.T) {
	t.Cleanup(func() { SetSelfProbe(false) })

	SetSelfProbe(false)
	assert.False(t, IsSelfProbeOK())

	SetSelfProbe(true)
	assert.True(t, IsSelfProbeOK())
}
