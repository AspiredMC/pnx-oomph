package world

import (
	"maps"
	"math"

	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/world"
)

// UnknownBlock is a block runtime ID that Oomph cannot resolve through its block registry.
// It is treated as a full solid block for simulation, but should not be sent back to the client.
type UnknownBlock struct {
	RuntimeID      uint32
	Name           string
	Properties     map[string]any
	NetworkHash    uint32
	HasNetworkHash bool
}

func NewUnknownBlock(runtimeID uint32) UnknownBlock {
	name, properties, _ := BlockRegistry.RuntimeIDToState(runtimeID)
	networkHash, hasNetworkHash := BlockRegistry.RuntimeIDToHash(runtimeID)
	return UnknownBlock{
		RuntimeID:      runtimeID,
		Name:           name,
		Properties:     maps.Clone(properties),
		NetworkHash:    networkHash,
		HasNetworkHash: hasNetworkHash,
	}
}

func (b UnknownBlock) EncodeBlock() (string, map[string]any) {
	if b.Name != "" {
		return b.Name, maps.Clone(b.Properties)
	}
	properties := map[string]any{"runtime_id": int32(b.RuntimeID)}
	if b.HasNetworkHash {
		properties["network_hash"] = int32(b.NetworkHash)
	}
	return "oomph:unknown", properties
}

func (b UnknownBlock) Hash() (uint64, uint64) {
	return 0, math.MaxUint64
}

func (b UnknownBlock) Model() world.BlockModel {
	return model.Solid{}
}

// BlockRuntimeID returns the stored runtime ID for unknown blocks and the registry runtime ID for known blocks.
func BlockRuntimeID(b world.Block) (uint32, bool) {
	if unknown, ok := b.(UnknownBlock); ok {
		return unknown.RuntimeID, false
	}
	return BlockRegistry.BlockRuntimeID(b), true
}
