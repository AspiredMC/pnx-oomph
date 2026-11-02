package world

import (
	"bytes"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	df_world "github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/oomph-ac/oomph/anticheat/world/blocknetwork"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestCacheChunkConvertsNetworkHashesToRuntimeIDs(t *testing.T) {
	FinalizeBlockRegistry()
	runtimeID, hash, source := newHashChunk(t)
	data := chunk.Encode(source, chunk.NetworkEncoding)
	input := &packet.LevelChunk{
		Dimension:     int32(packet.DimensionOverworld),
		SubChunkCount: uint32(len(data.SubChunks)),
		RawPayload:    serializedChunk(data),
	}

	info, err := CacheChunk(input, blocknetwork.NewCodec(BlockRegistry, blocknetwork.Hashes))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Chunk.Block(3, 64, 5, 0); got != runtimeID {
		t.Fatalf("cached block ID = %d, want runtime ID %d (network hash %d)", got, runtimeID, hash)
	}
}

func TestCacheSubChunkConvertsNetworkHashesToRuntimeIDs(t *testing.T) {
	FinalizeBlockRegistry()
	runtimeID, hash, source := newHashChunk(t)
	index := source.SubIndex(64)
	payload := bytes.NewBuffer(chunk.EncodeSubChunk(source, chunk.NetworkEncoding, int(index)))
	target := chunk.New(BlockRegistry, df_world.Overworld.Range())

	cached, err := CacheSubChunk(payload, target, [2]int32{}, blocknetwork.NewCodec(BlockRegistry, blocknetwork.Hashes))
	if err != nil {
		t.Fatal(err)
	}
	if got := cached.SubChunk().Block(3, 0, 5, 0); got != runtimeID {
		t.Fatalf("cached subchunk block ID = %d, want runtime ID %d (network hash %d)", got, runtimeID, hash)
	}
}

func TestReencodeLevelChunkPreservesNetworkHashes(t *testing.T) {
	FinalizeBlockRegistry()
	_, hash, source := newHashChunk(t)
	data := chunk.Encode(source, chunk.NetworkEncoding)
	input := &packet.LevelChunk{
		Dimension:     int32(packet.DimensionOverworld),
		SubChunkCount: uint32(len(data.SubChunks)),
		RawPayload:    serializedChunk(data),
	}
	codec := blocknetwork.NewCodec(BlockRegistry, blocknetwork.Hashes)

	if err := ReencodeLevelChunk(input, codec); err != nil {
		t.Fatal(err)
	}
	decoded, err := chunk.NetworkDecode(BlockRegistry, input.RawPayload, int(input.SubChunkCount), df_world.Overworld.Range())
	if err != nil {
		t.Fatal(err)
	}
	if got := decoded.Block(3, 64, 5, 0); got != hash {
		t.Fatalf("re-encoded block ID = %d, want network hash %d", got, hash)
	}
}

func newHashChunk(t *testing.T) (runtimeID, hash uint32, c *chunk.Chunk) {
	t.Helper()
	runtimeID = BlockRegistry.BlockRuntimeID(block.Stone{})
	var ok bool
	hash, ok = BlockRegistry.RuntimeIDToHash(runtimeID)
	if !ok {
		t.Fatal("stone runtime ID has no network hash")
	}
	c = chunk.New(BlockRegistry, df_world.Overworld.Range())
	c.SetBlock(3, 64, 5, 0, hash)
	return runtimeID, hash, c
}

func serializedChunk(data chunk.SerialisedData) []byte {
	var out bytes.Buffer
	for _, sub := range data.SubChunks {
		out.Write(sub)
	}
	out.Write(data.Biomes)
	return out.Bytes()
}
