package clightning

import (
	"errors"
	"testing"

	"github.com/elementsproject/peerswap/swap"
)

func TestLiquidSuspensionBeforeBackendAccess(t *testing.T) {
	// No CLN client or wallet: rejection must work even when the backend is down.
	for _, force := range []bool{false, true} {
		_, err := (&SwapIn{Asset: liquidAsset, Force: force}).Call()
		if !errors.Is(err, swap.ErrNewLiquidSwapsDisabled) {
			t.Fatalf("swap-in: %v", err)
		}
		_, err = (&SwapOut{Asset: liquidAsset, Force: force}).Call()
		if !errors.Is(err, swap.ErrNewLiquidSwapsDisabled) {
			t.Fatalf("swap-out: %v", err)
		}
	}
}
