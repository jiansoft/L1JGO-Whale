package handler

import (
	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/net/packet"
	"github.com/l1jgo/server/internal/world"
)

// NPC IDs that cannot equip pet items (Java hardcoded list in C_UsePetItem).
var petNoEquipNpcIDs = map[int32]bool{
	45034: true, 45039: true, 45040: true, 45042: true,
	45043: true, 45044: true, 45046: true, 45047: true,
	45048: true, 45049: true, 45053: true, 45054: true,
	45313: true, 46042: true, 45711: true, 46044: true,
}

// HandlePetMenu processes C_PETMENU (opcode 103).
// Java: C_PetMenu — client requests to view pet inventory/equipment.
func HandlePetMenu(sess *net.Session, r *packet.Reader, deps *Deps) {
	petID := r.ReadD()

	player := deps.World.GetBySession(sess.ID)
	if player == nil {
		return
	}

	pet := deps.World.GetPet(petID)
	if pet == nil || pet.OwnerCharID != player.CharID {
		return
	}

	sendPetInventory(sess, pet)
}

// HandleUsePetItem processes C_USEPETITEM (opcode 104).
// Java: C_UsePetItem — toggle equip/unequip pet weapon or armor.
func HandleUsePetItem(sess *net.Session, r *packet.Reader, deps *Deps) {
	_ = r.ReadC()      // data byte (always 0x00)
	petID := r.ReadD()  // pet object ID
	listNo := r.ReadC() // item index in pet's inventory

	player := deps.World.GetBySession(sess.ID)
	if player == nil {
		return
	}

	pet := deps.World.GetPet(petID)
	if pet == nil || pet.OwnerCharID != player.CharID {
		return
	}

	// Check if this pet type can equip items
	if petNoEquipNpcIDs[pet.NpcID] {
		sendPetInventory(sess, pet)
		return
	}
	petType := deps.PetTypes.Get(pet.NpcID)
	if petType != nil && !petType.CanEquip {
		sendPetInventory(sess, pet)
		return
	}

	// Look up item by index in pet's inventory
	idx := int(listNo)
	if idx < 0 || idx >= len(pet.Items) {
		sendPetInventory(sess, pet)
		return
	}
	item := pet.Items[idx]

	// Toggle equip/unequip
	if item.IsWeapon {
		if item.Equipped {
			unequipPetWeapon(pet, deps)
		} else {
			equipPetWeapon(pet, item, deps)
		}
	} else {
		if item.Equipped {
			unequipPetArmor(pet, deps)
		} else {
			equipPetArmor(pet, item, deps)
		}
	}

	// Send equipment update + refreshed inventory
	var equipMode byte
	if item.IsWeapon {
		equipMode = 1 // weapon
	} else {
		equipMode = 0 // armor
	}
	var equipStatus byte
	if item.Equipped {
		equipStatus = 1 // now equipped
	} else {
		equipStatus = 0 // now unequipped
	}
	sendPetEquipUpdate(sess, pet, equipMode, equipStatus)
	sendPetInventory(sess, pet)
}

// equipPetWeapon equips a weapon (tooth) on a pet, applying all stat bonuses.
// Java: C_UsePetItem.setPetWeapon — applies hit, dmg, and all stat modifiers.
func equipPetWeapon(pet *world.PetInfo, item *world.PetInvItem, deps *Deps) {
	if deps.PetItems == nil {
		return
	}
	info := deps.PetItems.Get(item.ItemID)
	if info == nil {
		return
	}

	// Unequip existing weapon first
	if pet.WeaponObjID != 0 {
		unequipPetWeapon(pet, deps)
	}

	pet.WeaponObjID = item.ObjectID
	item.Equipped = true

	// Apply combat bonuses
	pet.HitByWeapon = info.Hit
	pet.DamageByWeapon = info.Dmg

	// Apply stat bonuses
	pet.AddStr += info.AddStr
	pet.AddCon += info.AddCon
	pet.AddDex += info.AddDex
	pet.AddInt += info.AddInt
	pet.AddWis += info.AddWis
	pet.AddHP += info.AddHP
	pet.AddMP += info.AddMP
	pet.AddSP += info.AddSP
	pet.MDef += info.MDef
}

// unequipPetWeapon removes the equipped weapon from a pet, reverting all bonuses.
func unequipPetWeapon(pet *world.PetInfo, deps *Deps) {
	if pet.WeaponObjID == 0 {
		return
	}

	// Find the equipped weapon item
	var equippedItem *world.PetInvItem
	for _, it := range pet.Items {
		if it.ObjectID == pet.WeaponObjID && it.Equipped {
			equippedItem = it
			break
		}
	}

	// Reverse stat bonuses if we have the item info
	if equippedItem != nil && deps.PetItems != nil {
		info := deps.PetItems.Get(equippedItem.ItemID)
		if info != nil {
			pet.AddStr -= info.AddStr
			pet.AddCon -= info.AddCon
			pet.AddDex -= info.AddDex
			pet.AddInt -= info.AddInt
			pet.AddWis -= info.AddWis
			pet.AddHP -= info.AddHP
			pet.AddMP -= info.AddMP
			pet.AddSP -= info.AddSP
			pet.MDef -= info.MDef
		}
		equippedItem.Equipped = false
	}

	pet.HitByWeapon = 0
	pet.DamageByWeapon = 0
	pet.WeaponObjID = 0
}

// equipPetArmor equips armor on a pet, applying AC and stat bonuses.
// Java: C_UsePetItem.setPetArmor — applies AC and all stat modifiers.
func equipPetArmor(pet *world.PetInfo, item *world.PetInvItem, deps *Deps) {
	if deps.PetItems == nil {
		return
	}
	info := deps.PetItems.Get(item.ItemID)
	if info == nil {
		return
	}

	// Unequip existing armor first
	if pet.ArmorObjID != 0 {
		unequipPetArmor(pet, deps)
	}

	pet.ArmorObjID = item.ObjectID
	item.Equipped = true

	// Apply AC bonus (AC is negative = better, so subtract)
	pet.AC -= int16(info.AC)

	// Apply stat bonuses
	pet.AddStr += info.AddStr
	pet.AddCon += info.AddCon
	pet.AddDex += info.AddDex
	pet.AddInt += info.AddInt
	pet.AddWis += info.AddWis
	pet.AddHP += info.AddHP
	pet.AddMP += info.AddMP
	pet.AddSP += info.AddSP
	pet.MDef += info.MDef
}

// unequipPetArmor removes the equipped armor from a pet, reverting all bonuses.
func unequipPetArmor(pet *world.PetInfo, deps *Deps) {
	if pet.ArmorObjID == 0 {
		return
	}

	// Find the equipped armor item
	var equippedItem *world.PetInvItem
	for _, it := range pet.Items {
		if it.ObjectID == pet.ArmorObjID && it.Equipped {
			equippedItem = it
			break
		}
	}

	// Reverse stat bonuses
	if equippedItem != nil && deps.PetItems != nil {
		info := deps.PetItems.Get(equippedItem.ItemID)
		if info != nil {
			pet.AC += int16(info.AC) // reverse AC
			pet.AddStr -= info.AddStr
			pet.AddCon -= info.AddCon
			pet.AddDex -= info.AddDex
			pet.AddInt -= info.AddInt
			pet.AddWis -= info.AddWis
			pet.AddHP -= info.AddHP
			pet.AddMP -= info.AddMP
			pet.AddSP -= info.AddSP
			pet.MDef -= info.MDef
		}
		equippedItem.Equipped = false
	}

	pet.ArmorObjID = 0
}
