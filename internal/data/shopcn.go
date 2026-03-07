package data

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ShopCnItem 代表寄賣商城中的一個商品。
type ShopCnItem struct {
	ItemID          int32 `yaml:"item_id"`
	SellingPrice    int32 `yaml:"selling_price"`    // 售價（天寶幣）
	PurchasingPrice int32 `yaml:"purchasing_price"` // 回收基礎價
	PackCount       int32 `yaml:"pack_count"`       // 每包數量
	EnchantLevel    int16 `yaml:"enchant_level"`    // 強化等級
}

// ShopCnTable 儲存所有 NPC 的寄賣商城資料。
type ShopCnTable struct {
	byNpcID map[int32][]*ShopCnItem
	// 回收價查詢：itemID → 回收單價
	recyclePrice map[int32]int32
	// 所有商城物品 ID 集合（用於回收判斷）
	cnItemIDs map[int32]bool
}

type shopCnYAML struct {
	NpcID int32        `yaml:"npc_id"`
	Items []*ShopCnItem `yaml:"items"`
}

// LoadShopCn 從 YAML 檔案載入寄賣商城資料。
func LoadShopCn(dataDir string) (*ShopCnTable, error) {
	t := &ShopCnTable{
		byNpcID:      make(map[int32][]*ShopCnItem),
		recyclePrice: make(map[int32]int32),
		cnItemIDs:    make(map[int32]bool),
	}

	path := filepath.Join(dataDir, "shop_cn.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return t, nil // 檔案不存在時回傳空表
		}
		return nil, fmt.Errorf("讀取 shop_cn.yaml 失敗: %w", err)
	}

	var entries []shopCnYAML
	if err := yaml.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("解析 shop_cn.yaml 失敗: %w", err)
	}

	for _, e := range entries {
		t.byNpcID[e.NpcID] = e.Items
		for _, item := range e.Items {
			t.cnItemIDs[item.ItemID] = true
			// 計算回收價（Java: ShopCnTable 第 128-181 行）
			price := calcCnRecyclePrice(item)
			if price > 0 {
				t.recyclePrice[item.ItemID] = price
			}
		}
	}

	return t, nil
}

// calcCnRecyclePrice 計算商城物品回收單價。
// Java 邏輯：purchasingPrice > 0 → 用 purchasingPrice 或 sellingPrice/packCount/2；
// 否則用 sellingPrice/2 或 sellingPrice/packCount/2。
func calcCnRecyclePrice(item *ShopCnItem) int32 {
	if item.PurchasingPrice > 0 {
		if item.PackCount > 0 {
			p := item.SellingPrice / item.PackCount / 2
			if p < 1 {
				return 0
			}
			return p
		}
		return item.PurchasingPrice
	}
	if item.SellingPrice > 0 {
		if item.PackCount > 0 {
			p := item.SellingPrice / item.PackCount / 2
			if p < 1 {
				return 0
			}
			return p
		}
		return item.SellingPrice / 2
	}
	return 0
}

// Get 取得指定 NPC 的商城物品清單。
func (t *ShopCnTable) Get(npcID int32) []*ShopCnItem {
	if t == nil {
		return nil
	}
	return t.byNpcID[npcID]
}

// GetRecyclePrice 取得指定物品的回收單價（0 = 不可回收）。
func (t *ShopCnTable) GetRecyclePrice(itemID int32) int32 {
	if t == nil {
		return 0
	}
	return t.recyclePrice[itemID]
}

// IsCnItem 檢查物品是否為商城物品。
func (t *ShopCnTable) IsCnItem(itemID int32) bool {
	if t == nil {
		return false
	}
	return t.cnItemIDs[itemID]
}
