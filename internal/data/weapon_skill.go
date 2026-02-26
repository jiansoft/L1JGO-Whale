package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// WeaponSkillInfo 武器技能資料（weapon_skill 表）。
// 攻擊命中時以 Probability 機率觸發，造成額外傷害並播放 GFX。
type WeaponSkillInfo struct {
	WeaponID     int32  // 武器 item_id（主鍵）
	Note         string // 備註名稱
	Probability  int    // 觸發機率（1-100%）
	FixDamage    int    // 固定傷害
	RandomDamage int    // 隨機傷害上限（0 ~ 此值）
	Area         int    // AoE 範圍（0=單體, -1=全範圍, >0=格數）
	SkillID      int32  // 附加 buff 技能 ID（0=無）
	SkillTime    int    // buff 持續秒數
	EffectID     int32  // GFX 特效 ID（0=無特效）
	EffectTarget int    // 特效目標（0=敵人, 1=自己）
	ArrowType    bool   // 是否為投射物類型 GFX
	Attr         int    // 屬性（0=無, 1=地, 2=火, 4=水, 8=風）
}

// WeaponSkillTable 武器技能查找表，key = weapon item_id。
type WeaponSkillTable struct {
	skills map[int32]*WeaponSkillInfo
}

// Get 依武器 item_id 查詢武器技能，無則回傳 nil。
func (t *WeaponSkillTable) Get(weaponID int32) *WeaponSkillInfo {
	if t == nil {
		return nil
	}
	return t.skills[weaponID]
}

// Count 回傳已載入的武器技能數量。
func (t *WeaponSkillTable) Count() int {
	if t == nil {
		return 0
	}
	return len(t.skills)
}

// --- YAML 載入 ---

type weaponSkillEntry struct {
	WeaponID     int32  `yaml:"weapon_id"`
	Note         string `yaml:"note"`
	Probability  int    `yaml:"probability"`
	FixDamage    int    `yaml:"fix_damage"`
	RandomDamage int    `yaml:"random_damage"`
	Area         int    `yaml:"area"`
	SkillID      int32  `yaml:"skill_id"`
	SkillTime    int    `yaml:"skill_time"`
	EffectID     int32  `yaml:"effect_id"`
	EffectTarget int    `yaml:"effect_target"`
	ArrowType    bool   `yaml:"arrow_type"`
	Attr         int    `yaml:"attr"`
}

type weaponSkillFile struct {
	WeaponSkills []weaponSkillEntry `yaml:"weapon_skills"`
}

// LoadWeaponSkillTable 從 YAML 檔案載入武器技能資料。
func LoadWeaponSkillTable(path string) (*WeaponSkillTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("讀取武器技能資料失敗: %w", err)
	}

	var f weaponSkillFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("解析武器技能 YAML 失敗: %w", err)
	}

	t := &WeaponSkillTable{
		skills: make(map[int32]*WeaponSkillInfo, len(f.WeaponSkills)),
	}

	for i := range f.WeaponSkills {
		e := &f.WeaponSkills[i]
		t.skills[e.WeaponID] = &WeaponSkillInfo{
			WeaponID:     e.WeaponID,
			Note:         e.Note,
			Probability:  e.Probability,
			FixDamage:    e.FixDamage,
			RandomDamage: e.RandomDamage,
			Area:         e.Area,
			SkillID:      e.SkillID,
			SkillTime:    e.SkillTime,
			EffectID:     e.EffectID,
			EffectTarget: e.EffectTarget,
			ArrowType:    e.ArrowType,
			Attr:         e.Attr,
		}
	}

	return t, nil
}
