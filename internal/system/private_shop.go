package system

import (
	"fmt"

	"github.com/l1jgo/server/internal/handler"
	"github.com/l1jgo/server/internal/world"
)

// PrivateShopSystem 處理個人商店交易邏輯。
// 實作 handler.PrivateShopManager 介面。
type PrivateShopSystem struct {
	deps *handler.Deps
}

// NewPrivateShopSystem 建立個人商店交易系統。
func NewPrivateShopSystem(deps *handler.Deps) *PrivateShopSystem {
	return &PrivateShopSystem{deps: deps}
}

// SetupShop 開設個人商店（設定出售/收購清單 + 廣播擺攤動作）。
func (s *PrivateShopSystem) SetupShop(player *world.PlayerInfo, sellList []*world.PrivateShopSell, buyList []*world.PrivateShopBuy, shopChat []byte) {
	player.PrivateShop = true
	player.ShopSellList = sellList
	player.ShopBuyList = buyList
	player.ShopChat = shopChat

	nearby := s.deps.World.GetNearbyPlayersAt(player.X, player.Y, player.MapID)
	handler.BroadcastToPlayers(nearby, handler.BuildActionGfx(player.CharID, 3)) // 先取消原有動作
	shopData := handler.BuildShopAction(player.CharID, shopChat)
	handler.BroadcastToPlayers(nearby, shopData)
}

// CloseShop 關閉個人商店（清除狀態 + 廣播取消動作）。
func (s *PrivateShopSystem) CloseShop(player *world.PlayerInfo) {
	player.PrivateShop = false
	player.ShopSellList = nil
	player.ShopBuyList = nil
	player.ShopChat = nil
	player.ShopTradingLocked = false

	nearby := s.deps.World.GetNearbyPlayersAt(player.X, player.Y, player.MapID)
	handler.BroadcastToPlayers(nearby, handler.BuildActionGfx(player.CharID, 3))
}

// CancelShopNotTradable 因不可交易物品取消商店設置。
func (s *PrivateShopSystem) CancelShopNotTradable(player *world.PlayerInfo) {
	player.PrivateShop = false
	player.ShopSellList = nil
	player.ShopBuyList = nil

	nearby := s.deps.World.GetNearbyPlayersAt(player.X, player.Y, player.MapID)
	handler.BroadcastToPlayers(nearby, handler.BuildActionGfx(player.CharID, 3))
}

// ExecuteBuy 執行從個人商店購買物品（業務驗證 + 物品/金幣轉移 + 售完清理）。
func (s *PrivateShopSystem) ExecuteBuy(buyer *world.PlayerInfo, shopPlayer *world.PlayerInfo, orders []handler.ShopBuyOrder) {
	if shopPlayer.ShopTradingLocked {
		return
	}
	shopPlayer.ShopTradingLocked = true
	defer func() { shopPlayer.ShopTradingLocked = false }()

	sellList := shopPlayer.ShopSellList
	if len(sellList) == 0 {
		return
	}

	// 驗證商品數量一致性（Java: getPartnersPrivateShopItemCount）
	if buyer.ShopPartnerCount != len(sellList) {
		return
	}

	for _, o := range orders {
		if o.Order < 0 || o.Order >= len(sellList) || o.Count <= 0 {
			continue
		}

		pssl := sellList[o.Order]
		remaining := pssl.SellTotal - pssl.SoldCount
		count := o.Count
		if count > remaining {
			count = remaining
		}
		if count <= 0 {
			continue
		}

		item := shopPlayer.Inv.FindByObjectID(pssl.ItemObjectID)
		if item == nil {
			handler.SendServerMessage(buyer.Session, 989) // 無法交易
			continue
		}

		// 價格溢位保護（Java: price * count > 2000000000）
		totalPrice := int64(pssl.SellPrice) * int64(count)
		if totalPrice > 2_000_000_000 {
			handler.SendServerMessageArgs(buyer.Session, 904, "2000000000")
			return
		}
		price := int32(totalPrice)

		// 驗證買方金幣
		buyerAdena := buyer.Inv.GetAdena()
		if buyerAdena < price {
			handler.SendServerMessage(buyer.Session, 189) // 金幣不足
			continue
		}

		// 驗證賣方物品數量
		if item.Count < count {
			handler.SendServerMessage(buyer.Session, 989) // 無法交易
			continue
		}

		// 驗證買方背包容量
		if !item.Stackable && buyer.Inv.Size()+int(count) > world.MaxInventorySize {
			handler.SendServerMessage(buyer.Session, 270) // 背包過重
			break
		}

		// 執行物品轉移：賣方 → 買方
		s.TransferItem(shopPlayer, buyer, item, count)

		// 執行金幣轉移：買方 → 賣方
		s.TransferGold(buyer, shopPlayer, price)

		// 通知商店玩家：出售成功（Java: S_ServerMessage 877）
		itemName := item.Name
		if count > 1 {
			itemName = fmt.Sprintf("%s (%d)", item.Name, count)
		}
		handler.SendServerMessageArgs(shopPlayer.Session, 877, buyer.Name, itemName)

		// 更新售出累計
		pssl.SoldCount += count
	}

	// 清理已售完的項目（從末尾向前刪除）
	for i := len(sellList) - 1; i >= 0; i-- {
		if sellList[i].SoldCount >= sellList[i].SellTotal {
			sellList = append(sellList[:i], sellList[i+1:]...)
		}
	}
	shopPlayer.ShopSellList = sellList

	// 如果所有商品都售完，自動關閉商店
	if len(sellList) == 0 && len(shopPlayer.ShopBuyList) == 0 {
		s.CloseShop(shopPlayer)
	}
}

// ExecuteSell 執行向個人商店出售物品（業務驗證 + 物品/金幣轉移 + 收購完成清理）。
func (s *PrivateShopSystem) ExecuteSell(seller *world.PlayerInfo, shopPlayer *world.PlayerInfo, orders []handler.ShopSellOrder) {
	if shopPlayer.ShopTradingLocked {
		return
	}
	shopPlayer.ShopTradingLocked = true
	defer func() { shopPlayer.ShopTradingLocked = false }()

	buyList := shopPlayer.ShopBuyList
	if len(buyList) == 0 {
		return
	}

	for _, o := range orders {
		if o.Order < 0 || o.Order >= len(buyList) || o.Count <= 0 {
			continue
		}

		psbl := buyList[o.Order]
		remaining := psbl.BuyTotal - psbl.BoughtCount
		count := o.Count
		if count > remaining {
			count = remaining
		}
		if count <= 0 {
			continue
		}

		// 驗證玩家背包中的物品
		item := seller.Inv.FindByObjectID(o.ItemObjID)
		if item == nil {
			continue
		}

		// 驗證物品種類和強化等級匹配（防作弊）
		if item.ItemID != psbl.ItemID || item.EnchantLvl != psbl.EnchantLvl {
			return // 可能作弊
		}

		// 驗證物品數量
		if item.Count < count {
			handler.SendServerMessage(seller.Session, 989)
			continue
		}

		// 價格計算
		totalPrice := int64(psbl.BuyPrice) * int64(count)
		if totalPrice > 2_000_000_000 {
			handler.SendServerMessageArgs(seller.Session, 904, "2000000000")
			return
		}
		price := int32(totalPrice)

		// 驗證商店玩家金幣
		shopAdena := shopPlayer.Inv.GetAdena()
		if shopAdena < price {
			handler.SendServerMessage(seller.Session, 189) // 金幣不足
			break
		}

		// 驗證商店玩家背包容量
		if !item.Stackable && shopPlayer.Inv.Size()+int(count) > world.MaxInventorySize {
			handler.SendServerMessage(seller.Session, 271) // 對方背包過重
			break
		}

		// 執行物品轉移：賣方（玩家）→ 商店玩家
		s.TransferItem(seller, shopPlayer, item, count)

		// 執行金幣轉移：商店玩家 → 玩家
		s.TransferGold(shopPlayer, seller, price)

		// 更新收購累計
		psbl.BoughtCount += count
	}

	// 清理已收購完的項目
	for i := len(buyList) - 1; i >= 0; i-- {
		if buyList[i].BoughtCount >= buyList[i].BuyTotal {
			buyList = append(buyList[:i], buyList[i+1:]...)
		}
	}
	shopPlayer.ShopBuyList = buyList

	// 所有商品收購完成 → 自動關閉商店
	if len(shopPlayer.ShopSellList) == 0 && len(buyList) == 0 {
		s.CloseShop(shopPlayer)
	}
}

// TransferItem 從來源玩家背包移動物品到目標玩家背包。
func (s *PrivateShopSystem) TransferItem(from, to *world.PlayerInfo, item *world.InvItem, count int32) {
	info := s.deps.Items.Get(item.ItemID)

	if item.Stackable && item.Count > count {
		// 可堆疊物品：扣減來源數量 + 更新顯示
		item.Count -= count
		from.Dirty = true
		handler.SendItemCountUpdate(from.Session, item)

		// 目標：增加數量或新增
		destItem := to.Inv.AddItem(item.ItemID, count, item.Name, item.InvGfx, item.Weight, true, item.Bless)
		to.Dirty = true
		handler.SendAddItem(to.Session, destItem, info)
	} else {
		// 不可堆疊或全部移出：移除整個物品
		from.Inv.RemoveItem(item.ObjectID, count)
		from.Dirty = true
		handler.SendRemoveInventoryItem(from.Session, item.ObjectID)

		// 目標：新增物品（保留原有屬性）
		newItem := to.Inv.AddItemWithID(0, item.ItemID, count, item.Name, item.InvGfx, item.Weight, item.Stackable, item.Bless)
		newItem.EnchantLvl = item.EnchantLvl
		newItem.Identified = item.Identified
		newItem.UseType = item.UseType
		newItem.AttrEnchantKind = item.AttrEnchantKind
		newItem.AttrEnchantLevel = item.AttrEnchantLevel
		newItem.Durability = item.Durability
		to.Dirty = true
		handler.SendAddItem(to.Session, newItem, info)
	}
}

// TransferGold 轉移金幣。
func (s *PrivateShopSystem) TransferGold(from, to *world.PlayerInfo, amount int32) {
	// 從來源扣除
	fromAdena := from.Inv.FindByItemID(world.AdenaItemID)
	if fromAdena == nil {
		return
	}
	fromAdena.Count -= amount
	from.Dirty = true
	if fromAdena.Count <= 0 {
		from.Inv.RemoveItem(fromAdena.ObjectID, 0)
		handler.SendRemoveInventoryItem(from.Session, fromAdena.ObjectID)
	} else {
		handler.SendItemCountUpdate(from.Session, fromAdena)
	}

	// 給目標增加
	info := s.deps.Items.Get(world.AdenaItemID)
	toAdena := to.Inv.AddItem(world.AdenaItemID, amount, "金幣", 0, 0, true, 0)
	if info != nil {
		toAdena.InvGfx = info.InvGfx
		toAdena.Weight = info.Weight
		toAdena.Name = info.Name
	}
	to.Dirty = true
	handler.SendAddItem(to.Session, toAdena, info)
}
