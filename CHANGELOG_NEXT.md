# 待推送變更

## 批次 C — 裝備/寵物/料理修復

### C1. 變身時可正常裝備耳環
- `data/polymorph.go`: `IsArmorEquipable()` 未知裝備類型（如 earring）預設允許裝備，不再誤攔

### C3. 隱身斗篷穿脫觸發隱形效果
- `system/equip.go`: 穿上隱身斗篷（20077/120077）自動設定隱身、廣播移除；脫下後解除隱身、廣播重現
- 新增 `applyInvisCloak()` 方法

### C4. 寵物/召喚物安全區內不主動攻擊
- `system/companion_ai.go`: `summonScanForTarget()` 和 `petScanForTarget()` 加入 `IsSafetyZone` 檢查

### C5. 料理 buff 效果系統
- `system/item_use.go`: 新增 `cookingBuffMap`（Lv1-Lv4 共 35 種料理 → buff 映射）和 `applyCookingBuff()`
- 料理使用後除飽食度外，套用對應 buff（AC/HP/MP/MR/SP/HPR/MPR/元素抗性）
- 同時只能有一個料理 buff，新料理自動覆蓋舊 buff
- 發送屬性更新封包 + 料理圖示
- `world/state.go`: 新增 `CookingID` 欄位追蹤當前料理 buff
- `handler/cooking.go`: 匯出 `SendCookingIcon()`

### C6. 寵物改名路由修復
- `handler/attr.go`: 新增 C_Attr mode 325 處理（寵物改名 Yes/No + 新名稱輸入）
- `system/pet_mgr.go`: changename 動作時暫存 `player.TempID = pet.ID`
- `world/state.go`: 新增 `TempID` 欄位（暫存目標 ID）

## 批次 D — 寵物/武器/穿牆修復

### D1. 寵物/召喚物可被治療魔法補血
- `system/skill.go`: `executeBuffSkill()` 新增寵物/召喚物目標查找（在 NPC 和玩家之間）
- 新增 `healCompanion()` 方法：計算治療量、更新 HP、發送 HP 血條

### D2. 寵物死後屍體保留 + 復活
- `system/visibility.go`: 死亡寵物發送 `SendActionGfx(pet.ID, 8)` 顯示屍體
- `system/skill.go`: 復活術（61/75）可對死亡寵物施放，新增 `resurrectPet()` 方法
- `world/pet.go`: 新增 `PetRevive()` 方法（重新佔據地圖格）

### D3. 武器吸血/吸魔系統（資料驅動）
- `data/item.go`: `weaponEntry` + `ItemInfo` 新增 `dice_hp`/`sucking_hp`/`dice_mp`/`sucking_mp` 欄位
- `world/equipment.go`: `EquipStats` 新增 4 個吸取累計欄位
- `world/state.go`: `PlayerInfo` 新增 `DrainDiceHP`/`DrainSuckingHP`/`DrainDiceMP`/`DrainSuckingMP`
- `system/equip.go`: 裝備時通過 `calcEquipStats` 自動累加吸取值，脫下時自動扣除
- `system/combat.go`: 新增 `applyWeaponDrain()` — 近戰和遠程攻擊命中後機率觸發 HP/MP 吸取
- `weapon_list.yaml`: 瑪那魔杖(126) dice_mp=15/sucking_mp=3、鋼鐵瑪那魔杖(127) dice_mp=15/sucking_mp=5

### D4. 魔法攻擊視線（LOS）檢查
- `data/mapdata.go`: 新增 `IsArrowPassable()` 檢查 tile bit 2-3（箭矢通行）
- `data/mapdata.go`: 新增 `HasLineOfSight()` — 移植 Java glanceCheck（8 方向步進 + 箭矢通行判定，最多 15 步）
- `system/skill.go`: `executeAttackSkill()` 距離檢查後加入 LOS 檢查，失敗發送施法失敗訊息

### D5. NPC 遠程攻擊視線（LOS）檢查
- `system/npc_ai.go`: `npcRangedAttack()` 開頭加入 LOS 檢查
- `system/npc_ai.go`: `executeNpcSkill()` 魔法投射物部分加入 LOS 檢查
- `system/npc_ai.go`: 攻擊路由層：遠程 NPC LOS 失敗時嘗試移動靠近目標，而非原地空轉
