# 待推送變更

<!-- 每次修改程式碼時在此記錄，推送後清空 -->

## 架構優化第三輪 — Handler 層殘留違規修正

### privateshop.go 瘦化
- HandlePrivateShopBuy/HandlePrivateShopSell：業務邏輯（價格驗證、背包容量檢查、SoldCount/BoughtCount 管理、售完清理、自動關店）全部移入 PrivateShopSystem.ExecuteBuy/ExecuteSell
- HandleShop (openPrivateShop)：擺攤狀態設定移入 PrivateShopSystem.SetupShop
- closePrivateShop：移入 PrivateShopSystem.CloseShop
- Handler 現在只做封包解析 + 委派

### clan.go 修正
- player.Food/FoodFullTime 的直接修改從 handler 移入 ClanSystem.HealMember

### movement.go 修正
- player.Heading 直接修改改為透過 world.State.ChangePlayerHeading() 設定

### 新增/擴充介面
- PrivateShopManager：新增 SetupShop、CloseShop、CancelShopNotTradable、ExecuteBuy、ExecuteSell
- ShopBuyOrder/ShopSellOrder：封包解析結果結構
- world.State.ChangePlayerHeading：朝向更新方法
- handler.BuildShopAction：匯出擺攤動作封包建構函式

## 城堡／攻城戰系統 — 完整實作

### 城堡資料層
- 新增 `persist/migrations/026_castles.sql` — 城堡資料表（8 座城堡）
- 新增 `data/yaml/castles.yaml` — 城堡靜態地理資料（戰區、塔座標、內城地圖、城門 NPC）
- 新增 `data/castle.go` — CastleTable YAML 載入器
- 新增 `persist/castle_repo.go` — CastleRepo（LoadAll、SaveTaxRate、SavePublicMoney、ResetForWar）

### 城堡系統核心 (CastleSystem)
- 新增 `system/castle_sys.go` — 城堡管理系統
  - 啟動時從 DB 載入 8 座城堡（稅率、寶庫金幣、攻城戰時間）
  - GetCastleIDByNpcLocation：依 NPC 座標判斷所屬城堡
  - SetTaxRate / Deposit / Withdraw / AddPublicMoney：城堡經濟操作（WAL 寫入 DB）
  - GetTaxRate / IsWarNow / GetCastle：查詢介面
  - TransferCastle：城堡主權轉移（更新公會 has_castle + DB）
  - IsInWarArea：檢查座標是否在城堡戰區內

### 攻城戰排程器 (CastleWarTickSystem)
- Phase 3 PostUpdate 系統，每 tick 檢查攻城戰開始/結束時間
- startWar：設定 IsWarNow、稅率歸 10%、寶庫歸 0、生成戰爭旗幟、廣播開始訊息
- endWar：結束攻城戰、清除旗幟、廣播結束訊息、計算下次時間

### 宣戰系統 (WarSystem)
- 新增 `system/war_sys.go` — 戰爭管理系統
  - DeclareWar：宣戰（攻城戰/模擬戰），完整驗證（君主、等級、血盟、盟徽、已有城堡等）
  - SurrenderWar / CeaseWar：投降/休戰
  - WinCastleWar：進攻方獲勝，結束所有相關戰爭
  - CeaseCastleWar：防禦方獲勝（時間到）
  - IsWar / IsClanInWar / CheckCastleWar：查詢介面
  - 模擬戰 240 分鐘計時器（tick 內遞減）

### 城堡 NPC 動作
- handler/castle.go 新增 NPC 動作處理：inex、tax、withdrawal、cdeposit、castlegate、openigate、closeigate
- 封包：S_TaxRate(185)、S_Deposit(4)、S_Drawal(141)、S_CastleMaster(69)
- handler/castle.go 新增 C_War(227) handler：宣戰/投降/休戰
- handler/castle.go 新增 C_OPCODE_TAX(19)、C_OPCODE_DEPOSIT(56)、C_OPCODE_WITHDRAW(44) handler

### 商店稅收整合
- system/shop.go BuyFromNpc 新增稅金計算：城堡稅 + 國稅(→亞丁#7) + 戰爭稅(固定15%) + 迪亞得稅(→#8)
- 稅金自動分配到各城堡寶庫（AddPublicMoney）

### 戰爭實體
- 守護塔死亡 → 生成王冠 NPC（81125），通知附近玩家
- 王冠點擊 → 城堡主權轉移（HandleCrownClick：驗證君主/盟主/無城/距離/宣戰）
- CanDamageTower：塔攻擊前檢查（攻城中、已宣戰、亞丁副塔規則）
- SpawnWarFlags / ClearWarFlags：戰爭旗幟生成/清除

### 戰鬥整合
- combat.go 近戰/遠程攻擊前加入 L1Tower CanDamageTower 檢查
- combat.go handleNpcDeath 加入 L1Tower 死亡特殊處理（生成王冠、不給經驗/掉落/重生）

### NPC 動作整合
- npcaction.go 加入 L1Crown 點擊路由（→ HandleCrownClick）

### 登入整合
- enterworld.go 加入城主皇冠標誌（S_CastleMaster）和攻城戰進行中通知（CheckCastleWar）

### 新增 Opcodes
- S_OPCODE_CASTLEMASTER(69)、S_OPCODE_TAXRATE(185)、S_OPCODE_WAR(84)、S_OPCODE_WARTIME(231)、S_OPCODE_DRAWAL(141)

### 新增/擴充介面
- CastleManager：GetCastleIDByNpcLocation、GetTaxRate、SetTaxRate、Deposit、Withdraw、AddPublicMoney、IsWarNow、GetCastle、TransferCastle、IsInWarArea、OnTowerDeath、HandleCrownClick、CanDamageTower、SpawnWarFlags、ClearWarFlags
- WarManager：DeclareWar、SurrenderWar、CeaseWar、WinCastleWar、CeaseCastleWar、IsWar、IsClanInWar、CheckCastleWar

## 攻城戰補足功能 — 投石車 / 戰爭禮物 / 攻城時間

### 投石車系統（NPC 90327-90337）
- 新增 11 台投石車 NPC 到 `npc_list.yaml`（肯特 4 台、奇巖 4 台、妖魔 3 台）
- 新增炸彈道具（82500）到 `etcitem_list.yaml`
- `world/npc.go` NpcInfo 新增 ShellDamageTime / ShellSilenceTime 冷卻欄位
- `combat.go` isAttackableNpc 加入 L1Catapult、近戰/遠程加入 CanDamageCatapult 檢查
- `combat.go` handleNpcDeath 加入 L1Catapult 死亡處理（動畫 + 移除）
- `castle_sys.go` 新增：
  - SpawnCatapults / ClearCatapults：投石車生成/清除（啟動時自動生成、攻城結束後重生）
  - CanDamageCatapult：攻擊方/防守方權限驗證
  - HandleCatapultAction：砲彈發射邏輯（目標座標表、10 秒冷卻、消耗炸彈、範圍傷害/沉默）
  - IsCatapultAttacker：判斷投石車攻守方
  - catapultTargets：完整砲彈目標座標常量表（Java C_NPCAction.java:3636-3726）
- `npcaction.go` 新增 L1Catapult 動作處理：攻城狀態/君主/攻守方驗證 → HTML 對話 → 砲彈發射委派
- 普通砲彈：300 固定傷害，±2 格範圍，S_EffectLocation + S_DoActionGfx
- 沉默砲彈：15 秒沉默效果（player.Silenced + CatapultSilenceEnd 自動到期）

### 投石車沉默計時
- `world/state.go` PlayerInfo 新增 CatapultSilenceEnd 欄位
- `system/poison.go` 新增 TickCatapultSilence 函式（15 秒到期自動解除沉默）
- `system/buff_tick.go` 每 tick 呼叫 TickCatapultSilence

### 戰爭禮物系統
- 新增 `data/yaml/castle_war_gifts.yaml` — 8 座城堡的攻城戰禮物設定
- 新增 `data/war_gift.go` — WarGiftTable YAML 載入器
- `castle_sys.go` endWar 加入 distributeWarGifts：攻城結束時發放物品給守城方在線公會成員
- `context.go` Deps 新增 WarGifts 欄位

### 攻城戰時間查詢
- `npcaction.go` 新增 `case "askwartime"` → handleAskWarTime
- guardCastleMap：近衛兵 NPC ID → 城堡 ID + HTML ID 映射（12 個 NPC）
- makeWarTimeStrings 邏輯：格式化攻城時間為年/月/日/時/分字串（妖魔城 5 元素、其他 6 元素）
- `castle.go` 新增 SendWarTime（S_WarTime opcode 231）：1997/01/01 17:00 基點 + 182 分鐘時間單位

### CastleManager 介面擴充
- CanDamageCatapult、SpawnCatapults、ClearCatapults、HandleCatapultAction、IsCatapultAttacker

### main.go 整合
- 載入 castle_war_gifts.yaml + 設定 deps.WarGifts
- 啟動時自動生成 3 座城堡共 11 台投石車

## 隊伍 / GM 訊息 / 掉落分配 補齊

### 聊天組隊邀請
- `handler/party.go` HandlePartyControl 新增 action 3（聊天組隊邀請 /chatinvite）
- 路由至已存在的 PartySystem.ChatInvite()（Java: C_ChatParty case 3）

### GM 訊息
- `handler/broadcast.go` 新增 SendGmMessage / BroadcastToGMs
- 封包格式：S_OPCODE_NPCSHOUT(161), type=0, npcID=0, \fY 黃色文字（Java: S_ToGmMessage）
- `world/state.go` PlayerInfo 新增 AccessLevel 欄位
- `handler/enterworld.go` 進入世界時載入 AccessLevel

### 自動分配掉落模式
- `item_use.go` GiveDrops 擴充：自動分配隊伍（PartyTypeAutoShare）的掉落物按仇恨比例加權隨機分配
- 新增 collectAutoShareCandidates：收集同隊伍、同地圖、拾取範圍（15格）內的活人成員
- 新增 weightedRandomByHate：按 HateList 加權隨機選擇接收者（Java: DropShare.java）
- 新增 giveDropToPlayer：物品發放 + 封包通知（從 GiveDrops 提取的複用邏輯）
- GiveDrops 簽名改為傳入 NpcInfo（取得 HateList + NPC 座標）
- `combat.go` / `context.go` / `item.go` 同步更新呼叫簽名

### 傭兵系統
- Java C_HireSoldier / C_MercenaryArrange 均為空殼（未實作）
- Go handler/mercenary.go 空殼已正確處理（防止 unhandled opcode 日誌）
