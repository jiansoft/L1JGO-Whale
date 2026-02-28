# 更新日誌

## v0.3.0 (2026-02-28)

### 架構重構：Handler→System 大規模瘦身（Phase 4-11）

遵循「Handler 是薄層」架構原則，將業務邏輯從 handler/ 全面抽出至 system/。
Handler 只負責封包解析、基本驗證、委派到 System 或 Event Bus。

**新建 System 模組（13 個）：**
- `system/shop.go` — 商店買賣邏輯
- `system/weapon_skill.go` — 武器技能系統（揮空劍、骰子匕首）
- `system/craft.go` — 製作系統
- `system/item_ground.go` — 地面物品操作（銷毀、掉落、撿取）
- `system/pet_mgr.go` (910 行) — 寵物完整生命週期（召喚/收回/解放/死亡/經驗/裝備穿脫/馴服/進化）
- `system/doll.go` — 魔法娃娃召喚/解散/屬性加成
- `system/polymorph.go` — 變身系統（卷軸/技能驗證+消耗+執行）
- `system/death.go` — 死亡/重生系統
- `system/pvp.go` — PvP 戰鬥系統
- `system/mail.go` — 郵件系統
- `system/warehouse.go` — 倉庫系統
- `system/skill_summon.go` — 召喚獸系統
- `system/poison.go` — 毒系統

**Handler 瘦身成果：**

| 檔案 | 行數變化 | 削減率 |
|------|---------|--------|
| pet.go | 473 → 12 | -97% |
| doll.go | 219 → 5 | -98% |
| pet_tame.go | 329 → 72 | -78% |
| pet_inventory.go | 253 → 46 | -82% |
| polymorph.go | 243 → 147 | -40% |
| item.go | 973 → 779 | -20% |
| **總計** | **+1765 / -5651** | **淨減 3886 行** |

**介面驅動架構：**
- 每個 System 實作對應 Manager 介面（定義在 `handler/context.go`）
- 透過 `Deps` 結構注入，handler 呼叫 `deps.PetLife.UsePetCollar()` 等方法
- System 透過匯出的封包建構器（`handler.SendXxx()`）回送封包

### 程式碼清理

- 消除 `removeDollBonuses` 在 handler↔companion_ai 的重複定義
- 消除 `sendRemoveInvItem` 在 pet↔shop 的重複函式
- 統一 `petNoEquipNpcIDs` 至 system 層單一定義
- 移除所有已搬遷函式的殘留匯出包裝

---

## v0.2.1 (2026-02-27)

### 架構重構：Handler→System 分離（Phase 1-3）

Handler→System 架構分離的前 6 階段，system 佔比從 16% 提升至 37%。

**新建 System 模組（6 個）：**
- `TradeSystem` — 交易邏輯 + WAL 保護
- `PartySystem` — 組隊邀請/離開/驅逐/HP 同步
- `ClanSystem` — 血盟建立/加入/階級/盟徽
- `EquipSystem` — 裝備穿脫驗證 + 屬性計算
- `ItemUseSystem` — 藥水/卷軸/強化/技能書
- `SkillSystem 擴展` — 完整技能管線（35 → 1555 行）

### 倉庫系統擴充

- 角色專屬倉庫（每角色獨立）
- 血盟倉庫歷史記錄
- 延遲載入 + 多類型 DB 查詢鍵

### Bug 修復

- 修正輔助技能特效顯示在目標身上（C_UseSkill 封包分段讀取）
- 修正交易可交易性判斷邏輯
- NPC 中毒攻擊 + poison.go 獨立模組

---

## v0.2.0 (2026-02-24)

### 新增：實體碰撞系統
- 玩家無法穿越 NPC 和其他玩家，採用雙層碰撞機制
  - **客戶端層**：透過 S_CHANGE_ATTR (opcode 209) 封鎖格子四個方向的通行
  - **伺服端層**：EntityGrid 佔位檢查攔截對角線移動的穿越漏洞
  - 近距觸發：僅當玩家與實體距離 ≤ 3 格時才發送碰撞封包（避免遠距離 NPC 渲染消失）
  - 實體死亡自動解除碰撞格子
- 新增 `EntityGrid` 格子佔位表，O(1) 查詢碰撞
- 新增 `NpcAOIGrid` NPC 空間索引（取代 O(N) 全量掃描）

### 新增：警衛 AI 系統
- 警衛 (`L1Guard`) 主動追殺通緝犯/紅名玩家，被打時反擊
- 玩家 PK 後獲得 24 小時通緝狀態 (`WantedTicks`)
- 粉名觸發時自動通知附近警衛仇恨該玩家
- 追擊範圍 30 格，超過 30 格自動傳送回出生點
- 擊殺警衛不給經驗、正義值、掉落物

### 新增：Buff 持久化系統
- Buff 在登出/定期存檔時寫入資料庫，登入時還原
- 新增 `persist/buff_repo.go`，包含 `LoadByCharID` / `SaveBuffs`
- 新增資料庫遷移 `009_character_buffs.sql`
- 變身狀態可跨登出保留（登入時靜默還原外觀）
- 藥水 buff（勇敢、加速、慎重、藍水）持久化，剩餘時間準確還原
- 登入完成後補發所有 buff 圖示

### 新增：門系統
- 載入 `door_gfx.yaml` + `door_spawn.yaml` 門資料
- 伺服器啟動時生成門，進入世界時發送給玩家
- 新增 `world/door.go` (`DoorInfo`) 和 `handler/door.go` (`SendDoorPerceive`)

### 新增：傳送魔法系統
- 實作技能 5（瞬間移動）和技能 69（集體瞬間移動）
- 書籤傳送：檢查地圖是否允許逃脫 (`Escapable`)
- 隨機傳送：檢查地圖是否允許傳送 (`Teleportable`)，200 格半徑，40 次嘗試尋找可行點
- 失敗時不消耗 MP（與 Java 行為一致）
- 傳送時自動取消交易

### 改進：NPC AI 尋路
- 三層移動邏輯，防止怪物永久卡死：
  1. 直線朝目標移動
  2. 被擋時嘗試兩個側向方向繞路
  3. 全部被擋時強制穿過（僅檢查原始地形，忽略 NPC 動態佔位）
- NPC 之間不互相碰撞（與 Java 一致，怪物可自由重疊）
- NPC 重生時若出生點被佔用，自動搜尋附近空格

### 改進：死亡系統
- **死亡時清除所有 buff**（好壞都清，無例外），包括變身還原、加速/勇敢重置、所有屬性增減還原
- **死亡經驗懲罰從 10% 改為 5%**（當前等級經驗區間的 5%）
- 死亡正確釋放實體佔位格子，解除附近玩家的碰撞封包
- 重新開始（復活）時在新位置發送碰撞資料

### 改進：衝裝系統
- 衝裝卷軸不再要求裝備中才能使用
- 封印物品（bless ≥ 128）無法衝裝
- 支援負數衝裝等級（詛咒卷軸）
- 衝裝成功/失敗/爆裝訊息使用正確的 S_ServerMessage (161/160/164) 含顏色代碼
- 武器衝裝加成套用到命中/傷害計算 (`CalcEquipStats`)
- 詛咒卷軸機制：成功 = 衝裝 -1，失敗且 ≤ -7 時爆裝
- 祝福卷軸：失敗時無變化（受保護）

### 改進：物品系統
- `InvItem.EnchantLvl` 型別從 `byte` 改為 `int8`（支援負數衝裝）
- 負數衝裝等級正確顯示名稱（如「-3 乙武器」）
- 新增回家卷軸功能（傳送至復活點）
- 新增固定目的地傳送卷軸（使用 etcitem YAML 的 `LocX`/`LocY`）
- 新增藍色藥水 buff（MP 回復加速 + 圖示）
- 修正撿取物品時使用衝裝等級而非 bless 值的問題

### 改進：技能/Buff
- 暴風疾走（技能 172）現在正確建立 ActiveBuff 條目以支援持久化
- 暴風疾走施放前移除衝突的勇敢/速度 buff
- 慎重藥水和藍色藥水圖示正確發送/還原
- 藥水虛擬 SkillID (1000-1027) 整合到 buff 圖示系統
- 傳送魔法在 MP 消耗前先路由（失敗不扣 MP）

### 改進：PK 系統
- 粉名觸發時通知附近警衛仇恨攻擊者
- PK 殺人後設置 24 小時通緝狀態，每 tick 遞減

### 改進：GM 指令
- `.wall [1-5]` — 測試碰撞牆壁模式
- `.clearwall` — 清除測試牆壁
- `.hp` 和 `.heal` 現在正確處理復活（重置移動時間戳、重新佔位）
- `.kill` 和 `.killall` 現在正確從 AOI/實體格子移除 NPC
- `.item` 支援負數衝裝等級（-7 到 +15）

### 基礎設施
- `EntityGrid` 格子佔位追蹤（支援同格多實體）
- `NpcAOIGrid` NPC 空間查詢 O(cells)
- `State.UpdateNpcPosition()` — 集中化 NPC 位置更新（同步 AOI + 實體格子）
- `State.NpcDied()` / `State.NpcRespawn()` — NPC 死亡/重生時正確清理/恢復格子
- `IsPassableIgnoreOccupant()` — 忽略 NPC 動態佔位旗標的地形通行檢查（NPC 尋路保險用）
- 新增 opcode `S_OPCODE_CLANATTENTION` (200)
- `BuffRepo`、`DoorTable` 加入 handler Deps 依賴注入

### Bug 修復
- 修正 NPC 死亡後未從 AOI/實體格子移除（導致幽靈碰撞）
- 修正 NPC 在被佔用的格子重生（現在搜尋附近空格）
- 修正實體碰撞導致 NPC AI 永久卡住（三層尋路解決）
- 修正重新開始後新位置缺少碰撞資料
- 修正衝裝卷軸消耗後未更新負重顯示
- 修正撿取物品時 bless 欄位使用了衝裝等級而非物品本身的 bless 值
