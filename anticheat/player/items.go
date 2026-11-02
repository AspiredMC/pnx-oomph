package player

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/oomph-ac/oomph/anticheat/utils"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func (p *Player) ConvertToStack(it protocol.ItemStack) item.Stack {
	if hasCustomEnchantNBT(it.NBTData) {
		p.CustomEnchantExempt = true
	}

	t, ok := p.items[int16(it.ItemType.NetworkID)]
	if !ok {
		t, ok = world.ItemByRuntimeID(it.NetworkID, int16(it.MetadataValue))
		if !ok {
			t = block.Air{}
		}
	}
	if it.BlockRuntimeID != 0 {
		if b, found := p.World().BlockRegistry().BlockByRuntimeID(p.DecodeBlockRuntimeID(uint32(it.BlockRuntimeID))); found {
			if t, ok = b.(world.Item); !ok {
				t = block.Air{}
			}
		}
	}
	if nbter, ok := t.(world.NBTer); ok && len(it.NBTData) != 0 {
		t = nbter.DecodeNBT(it.NBTData).(world.Item)
	}
	s := item.NewStack(t, int(it.Count))
	return item.ReadNBT(it.NBTData, &s).AsUnbreakable()
}

func (p *Player) InstanceFromItem(it item.Stack) protocol.ItemInstance {
	instance := utils.InstanceFromItem(p.World().BlockRegistry(), it)
	if instance.Stack.BlockRuntimeID != 0 {
		instance.Stack.BlockRuntimeID = int32(p.EncodeBlockRuntimeID(uint32(instance.Stack.BlockRuntimeID)))
	}
	return instance
}

func (p *Player) StackToItem(it protocol.ItemStack) item.Stack {
	if it.BlockRuntimeID != 0 {
		it.BlockRuntimeID = int32(p.DecodeBlockRuntimeID(uint32(it.BlockRuntimeID)))
	}
	return utils.StackToItem(p.World().BlockRegistry(), it)
}

func hasCustomEnchantNBT(data map[string]any) bool {
	return scanCustomEnchantNBT(data, "")
}

func scanCustomEnchantNBT(value any, parentKey string) bool {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			lowerKey := strings.ToLower(key)
			if strings.Contains(lowerKey, "custom") && strings.Contains(lowerKey, "enchant") {
				return true
			}
			if isCustomEnchantValue(lowerKey, child) || scanCustomEnchantNBT(child, lowerKey) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if scanCustomEnchantNBT(child, parentKey) {
				return true
			}
		}
	case []map[string]any:
		for _, child := range v {
			if scanCustomEnchantNBT(child, parentKey) {
				return true
			}
		}
	}
	return false
}

func isCustomEnchantValue(key string, value any) bool {
	if !strings.Contains(key, "ench") {
		return false
	}

	switch v := value.(type) {
	case string:
		lower := strings.ToLower(v)
		return strings.Contains(lower, ":") && !strings.HasPrefix(lower, "minecraft:")
	case map[string]any:
		for childKey, child := range v {
			lowerChildKey := strings.ToLower(childKey)
			if strings.Contains(lowerChildKey, "custom") || strings.Contains(lowerChildKey, "namespace") || strings.Contains(lowerChildKey, "name") {
				return true
			}
			if lowerChildKey == "id" {
				if id, ok := child.(string); ok && strings.Contains(id, ":") && !strings.HasPrefix(strings.ToLower(id), "minecraft:") {
					return true
				}
			}
		}
	case []any:
		for _, child := range v {
			if isCustomEnchantValue(key, child) {
				return true
			}
		}
	default:
		text := strings.ToLower(fmt.Sprint(v))
		return strings.Contains(text, "custom") && strings.Contains(text, "enchant")
	}
	return false
}
