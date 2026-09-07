package test

import (
	"context"
	"strings"
	"testing"

	"github.com/elementsproject/peerswap/clightning"
	"github.com/elementsproject/peerswap/peerswaprpc"
	"github.com/elementsproject/peerswap/swap"
	"github.com/elementsproject/peerswap/testframework"
)

// During the suspension, the Liquid matrix checks rejection instead of
// attempting to create new contracts. Persisted recovery is tested separately.
func runLiquidSuspension(t *testing.T, vector string) {
	t.Helper()
	IsIntegrationTest(t)
	lwk := strings.HasPrefix(vector, "lwk_")
	if lwk {
		skipLWKTests(t)
	}
	var nodes []testframework.LightningNode
	var daemons []*PeerSwapd
	switch {
	case strings.HasSuffix(vector, "clncln"):
		if lwk {
			_, _, clns, _, _, _ := clnclnLWKSetup(t, 1000000)
			for _, n := range clns {
				nodes = append(nodes, n)
			}
		} else {
			_, _, clns, _ := clnclnElementsSetup(t, 1000000)
			for _, n := range clns {
				nodes = append(nodes, n)
			}
		}
	case strings.HasSuffix(vector, "lndlnd"):
		if lwk {
			_, _, lnds, ps, _, _, _ := lndlndLWKSetup(t, 1000000)
			daemons = ps
			for _, n := range lnds {
				nodes = append(nodes, n)
			}
		} else {
			_, _, lnds, ps, _ := lndlndElementsSetup(t, 1000000)
			daemons = ps
			for _, n := range lnds {
				nodes = append(nodes, n)
			}
		}
	case strings.HasSuffix(vector, "mixed"):
		if lwk {
			_, _, ns, ps, _, _, _ := mixedLWKSetup(t, 1000000, FunderCLN)
			nodes, daemons = ns, []*PeerSwapd{ps}
		} else {
			_, _, ns, ps, _ := mixedElementsSetup(t, 1000000, FunderCLN)
			nodes, daemons = ns, []*PeerSwapd{ps}
		}
	default:
		t.Fatalf("unknown Liquid test vector: %s", vector)
	}
	for _, node := range nodes {
		cln, ok := node.(*CLightningNodeWithLiquid)
		if !ok {
			continue
		}
		var result interface{}
		err := cln.Rpc.Request(&clightning.SwapIn{Asset: "lbtc", SatAmt: 100000, ShortChannelId: "1x1x1"}, &result)
		assertSuspended(t, err)
		err = cln.Rpc.Request(
			&clightning.SwapOut{Asset: "lbtc", SatAmt: 100000, ShortChannelId: "1x1x1", Force: true},
			&result,
		)
		assertSuspended(t, err)
	}
	for _, ps := range daemons {
		_, err := ps.PeerswapClient.SwapIn(
			context.Background(),
			&peerswaprpc.SwapInRequest{Asset: "lbtc", SwapAmount: 100000, ChannelId: 1},
		)
		assertSuspended(t, err)
		_, err = ps.PeerswapClient.SwapOut(
			context.Background(),
			&peerswaprpc.SwapOutRequest{Asset: "lbtc", SwapAmount: 100000, ChannelId: 1, Force: true},
		)
		assertSuspended(t, err)
	}
}

func assertSuspended(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), swap.ErrNewLiquidSwapsDisabled.Error()) {
		t.Fatalf("got %v, want Liquid suspension error", err)
	}
}
