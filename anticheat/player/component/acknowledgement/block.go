package acknowledgement

import (
	df_cube "github.com/df-mc/dragonfly/server/block/cube"
	"github.com/oomph-ac/oomph/anticheat/player"
	oworld "github.com/oomph-ac/oomph/anticheat/world"
)

type UpdateBlockBatch struct {
	mPlayer *player.Player
	updates map[df_cube.Pos]uint32

	expiresIn int64
	valid     bool
}

func NewUpdateBlockBatchACK(p *player.Player) *UpdateBlockBatch {
	return &UpdateBlockBatch{mPlayer: p, updates: make(map[df_cube.Pos]uint32), valid: true}
}

func (ack *UpdateBlockBatch) Blocks() map[df_cube.Pos]uint32 {
	return ack.updates
}

func (ack *UpdateBlockBatch) SetBlock(pos df_cube.Pos, b uint32) {
	ack.updates[pos] = b
}

func (ack *UpdateBlockBatch) RemoveBlock(pos df_cube.Pos) {
	delete(ack.updates, pos)
}

func (ack *UpdateBlockBatch) SetExpiry(expiresIn int64) {
	ack.expiresIn = expiresIn
}

func (ack *UpdateBlockBatch) HasUpdates() bool {
	return len(ack.updates) > 0
}

func (ack *UpdateBlockBatch) Run() {
	if !ack.valid {
		return
	}
	ack.valid = false
	for pos, bRuntimeID := range ack.updates {
		b, ok := oworld.BlockRegistry.BlockByRuntimeID(bRuntimeID)
		if !ok {
			b = oworld.NewUnknownBlock(bRuntimeID)
		}
		ack.mPlayer.World().SetBlock(pos, b, nil)
		ack.mPlayer.WorldUpdater().RemovePendingUpdate(pos, bRuntimeID)
	}
	ack.updates = nil
}

func (ack *UpdateBlockBatch) Tick() {
	if !ack.valid {
		return
	}
	ack.expiresIn--
	if ack.expiresIn <= 0 {
		ack.Run()
	}
}

func (ack *UpdateBlockBatch) Invalidate() {
	ack.valid = false
	ack.mPlayer.Dbg.Notify(player.DebugModeChunks, true, "updateBlockBatch ack is invalidated")
}
