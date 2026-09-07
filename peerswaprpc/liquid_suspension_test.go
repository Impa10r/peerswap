package peerswaprpc

import (
	"context"
	"errors"
	"testing"

	"github.com/elementsproject/peerswap/swap"
)

func TestLiquidSuspensionBeforeBackendAccess(t *testing.T) {
	// No LND client or wallet: rejection must not require a healthy backend.
	server := &PeerswapServer{}
	for _, force := range []bool{false, true} {
		_, err := server.SwapIn(context.Background(), &SwapInRequest{Asset: liquidAsset, Force: force})
		if !errors.Is(err, swap.ErrNewLiquidSwapsDisabled) {
			t.Fatalf("swap-in: %v", err)
		}
		_, err = server.SwapOut(context.Background(), &SwapOutRequest{Asset: liquidAsset, Force: force})
		if !errors.Is(err, swap.ErrNewLiquidSwapsDisabled) {
			t.Fatalf("swap-out: %v", err)
		}
	}
}
