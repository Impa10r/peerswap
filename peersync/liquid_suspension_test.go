package peersync

import (
	"reflect"
	"testing"
)

func TestSuspensionOmitsLiquidFromLocalCapability(t *testing.T) {
	ps, _ := newTestPeerSync(t)
	for _, assets := range [][]Asset{{AssetBTC, AssetLBTC}, {AssetLBTC}, {AssetBTC}} {
		ps.supportedAssets = assets
		capability := ps.localCapabilityForPeer(PeerID{})
		for _, asset := range capability.SupportedAssets() {
			if asset == AssetLBTC {
				t.Fatal("advertised suspended L-BTC")
			}
		}
		if !reflect.DeepEqual(assets, ps.supportedAssets) {
			t.Fatal("changed backend assets")
		}
		if len(assets) == 1 && assets[0] == AssetLBTC && len(capability.SupportedAssets()) != 0 {
			t.Fatal("Liquid-only node must advertise no available assets")
		}
	}
}
