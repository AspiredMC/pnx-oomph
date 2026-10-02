package world

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	df_world "github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestBlockUsesLocalChunkCoordinates(t *testing.T) {
	FinalizeBlockRegistry()

	w := New(nil)
	c := chunk.New(BlockRegistry, df_world.Overworld.Range())
	stairs := block.Stairs{
		Block:  block.Stone{},
		Facing: cube.North,
	}
	c.SetBlock(4, 64, 5, 0, BlockRegistry.BlockRuntimeID(stairs))

	w.AddChunk(protocol.ChunkPos{1, 0}, ChunkInfo{Chunk: c})
	got := w.Block(cube.Pos{20, 64, 5})
	if got != stairs {
		t.Fatalf("Block() = %#v, want %#v", got, stairs)
	}
}

func TestUnknownRuntimeIDIsSolidUnknownBlock(t *testing.T) {
	FinalizeBlockRegistry()

	w := New(nil)
	c := chunk.New(BlockRegistry, df_world.Overworld.Range())
	unknownRuntimeID := uint32(BlockRegistry.BlockCount() + 50)
	c.SetBlock(1, 64, 1, 0, unknownRuntimeID)

	w.AddChunk(protocol.ChunkPos{0, 0}, ChunkInfo{Chunk: c})
	got, ok := w.Block(cube.Pos{1, 64, 1}).(UnknownBlock)
	if !ok {
		t.Fatalf("Block() = %#v, want UnknownBlock", got)
	}
	if got.RuntimeID != unknownRuntimeID {
		t.Fatalf("UnknownBlock runtime ID = %d, want %d", got.RuntimeID, unknownRuntimeID)
	}
	if boxes := got.Model().BBox(cube.Pos{1, 64, 1}, w); len(boxes) != 1 || boxes[0].Height() != 1 || boxes[0].Width() != 1 || boxes[0].Length() != 1 {
		t.Fatalf("UnknownBlock model boxes = %#v, want one full block", boxes)
	}
	name, properties := got.EncodeBlock()
	if name != "oomph:unknown" || properties["runtime_id"] != int32(unknownRuntimeID) {
		t.Fatalf("UnknownBlock EncodeBlock() = %q, %#v", name, properties)
	}
}
