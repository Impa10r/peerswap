package swap

import (
	"errors"
	"fmt"
)

// ErrNewLiquidSwapsDisabled applies only to new swaps. Liquid services must
// remain enabled so that persisted swaps can still be recovered.
var ErrNewLiquidSwapsDisabled = errors.New("new L-BTC swaps are disabled due to the Liquid network security incident")

func (s *SwapService) rejectNewLiquidSwap(id *SwapId, peer string) error {
	if id == nil {
		return errors.New("missing swap id")
	}
	// Do not send a cancellation for a retransmission of an existing swap.
	if existing, err := s.GetActiveSwap(id.String()); err == nil {
		if existing.Data.PeerNodeId != peer {
			return ErrReceivedMessageFromUnexpectedPeer(peer, id)
		}
		return AlreadyExistsError
	}
	payload, kind, err := MarshalPeerswapMessage(&CancelMessage{
		SwapId:  id,
		Message: ErrNewLiquidSwapsDisabled.Error(),
	})
	if err != nil {
		return err
	}
	if err := s.swapServices.messenger.SendMessage(peer, payload, kind); err != nil {
		return fmt.Errorf("%w: sending cancellation: %w", ErrNewLiquidSwapsDisabled, err)
	}
	return ErrNewLiquidSwapsDisabled
}
