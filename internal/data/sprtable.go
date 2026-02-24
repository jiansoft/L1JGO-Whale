package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Action ID constants (ported from Java ActionCodes.java)
const (
	ActWalk                = 0
	ActAttack              = 1
	ActSwordWalk           = 4
	ActSwordAttack         = 5
	ActAxeWalk             = 11
	ActAxeAttack           = 12
	ActSkillAttack         = 18 // directional spell cast
	ActSkillBuff           = 19 // non-directional spell cast
	ActBowWalk             = 20
	ActBowAttack           = 21
	ActSpearWalk           = 24
	ActSpearAttack         = 25
	ActAltAttack           = 30
	ActSpellDirExtra       = 31
	ActStaffWalk           = 40
	ActStaffAttack         = 41
	ActDaggerWalk          = 46
	ActDaggerAttack        = 47
	ActTwoHandSwordWalk    = 50
	ActTwoHandSwordAttack  = 51
	ActEdoryuWalk          = 54
	ActEdoryuAttack        = 55
	ActClawWalk            = 58
	ActClawAttack          = 59
	ActThrowingKnifeWalk   = 62
	ActThrowingKnifeAttack = 63
	ActThink               = 66
	ActAggress             = 67
)

// walkActions and attackActions are lookup sets for categorising act_id.
var walkActions = map[int]bool{
	ActWalk: true, ActSwordWalk: true, ActAxeWalk: true,
	ActBowWalk: true, ActSpearWalk: true, ActStaffWalk: true,
	ActDaggerWalk: true, ActTwoHandSwordWalk: true, ActEdoryuWalk: true,
	ActClawWalk: true, ActThrowingKnifeWalk: true,
}

var attackActions = map[int]bool{
	ActAttack: true, ActSwordAttack: true, ActAxeAttack: true,
	ActBowAttack: true, ActSpearAttack: true, ActAltAttack: true,
	ActSpellDirExtra: true, ActStaffAttack: true, ActDaggerAttack: true,
	ActTwoHandSwordAttack: true, ActEdoryuAttack: true, ActClawAttack: true,
	ActThrowingKnifeAttack: true,
}

// sprData holds per-sprite animation timing data.
type sprData struct {
	moveSpeed       map[int]int // actID → ms
	attackSpeed     map[int]int // actID → ms
	specialSpeed    map[int]int // actID → ms (Think / Aggress)
	dirSpellSpeed   int         // ACTION_SkillAttack (18), default 1200
	nodirSpellSpeed int         // ACTION_SkillBuff (19), default 1200
}

func newSprData() *sprData {
	return &sprData{
		moveSpeed:       make(map[int]int),
		attackSpeed:     make(map[int]int),
		specialSpeed:    make(map[int]int),
		dirSpellSpeed:   1200,
		nodirSpellSpeed: 1200,
	}
}

// SprTable maps sprite IDs to animation timing data.
// Ported from Java SprTable.java.
type SprTable struct {
	data map[int]*sprData
}

// --- YAML loading ---

type sprActionEntry struct {
	SprID      int `yaml:"spr_id"`
	ActID      int `yaml:"act_id"`
	FrameCount int `yaml:"framecount"`
	FrameRate  int `yaml:"framerate"`
}

type sprActionFile struct {
	Actions []sprActionEntry `yaml:"spr_actions"`
}

// LoadSprTable loads sprite animation timing data from YAML.
// Formula (ported from Java): ms = frameCount * 40 * (24.0 / frameRate)
func LoadSprTable(path string) (*SprTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("sprtable: read %s: %w", path, err)
	}

	var f sprActionFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("sprtable: parse %s: %w", path, err)
	}

	t := &SprTable{data: make(map[int]*sprData, 4096)}

	for _, e := range f.Actions {
		if e.FrameRate <= 0 {
			continue
		}

		// Java formula: frameCount * 40 * (24.0 / frameRate)
		speedMs := int(float64(e.FrameCount) * 40.0 * (24.0 / float64(e.FrameRate)))

		spr, ok := t.data[e.SprID]
		if !ok {
			spr = newSprData()
			t.data[e.SprID] = spr
		}

		switch {
		case walkActions[e.ActID]:
			spr.moveSpeed[e.ActID] = speedMs
		case attackActions[e.ActID]:
			spr.attackSpeed[e.ActID] = speedMs
		case e.ActID == ActSkillAttack:
			spr.dirSpellSpeed = speedMs
		case e.ActID == ActSkillBuff:
			spr.nodirSpellSpeed = speedMs
		case e.ActID == ActThink || e.ActID == ActAggress:
			spr.specialSpeed[e.ActID] = speedMs
		}
	}

	return t, nil
}

// Count returns the number of unique sprite IDs loaded.
func (t *SprTable) Count() int {
	return len(t.data)
}

// GetAttackSpeed returns the attack animation duration (ms) for the given
// sprite and weapon action ID.
// Falls back to ACTION_Attack (1) when the specific weapon has no entry.
// Returns 0 if the sprite is unknown.
func (t *SprTable) GetAttackSpeed(sprID, actID int) int {
	spr, ok := t.data[sprID]
	if !ok {
		return 0
	}
	if v, ok := spr.attackSpeed[actID]; ok {
		return v
	}
	if actID == ActAttack {
		return 0
	}
	return spr.attackSpeed[ActAttack]
}

// GetMoveSpeed returns the walk animation duration (ms) for the given
// sprite and walk action ID.
// Falls back to ACTION_Walk (0) when the specific stance has no entry.
// Returns 0 if the sprite is unknown.
func (t *SprTable) GetMoveSpeed(sprID, actID int) int {
	spr, ok := t.data[sprID]
	if !ok {
		return 0
	}
	if v, ok := spr.moveSpeed[actID]; ok {
		return v
	}
	if actID == ActWalk {
		return 0
	}
	return spr.moveSpeed[ActWalk]
}

// GetDirSpellSpeed returns the directional spell cast animation duration (ms).
// Returns 0 if the sprite is unknown.
func (t *SprTable) GetDirSpellSpeed(sprID int) int {
	if spr, ok := t.data[sprID]; ok {
		return spr.dirSpellSpeed
	}
	return 0
}

// GetNodirSpellSpeed returns the non-directional spell cast animation duration (ms).
// Returns 0 if the sprite is unknown.
func (t *SprTable) GetNodirSpellSpeed(sprID int) int {
	if spr, ok := t.data[sprID]; ok {
		return spr.nodirSpellSpeed
	}
	return 0
}

// GetSpecialSpeed returns the Think/Aggress emote animation duration (ms).
// Returns 1200 ms as default when the action is missing; 0 when sprite unknown.
func (t *SprTable) GetSpecialSpeed(sprID, actID int) int {
	spr, ok := t.data[sprID]
	if !ok {
		return 0
	}
	if v, ok := spr.specialSpeed[actID]; ok {
		return v
	}
	return 1200
}

// GetSprSpeed is the unified dispatcher — mirrors Java SprTable.getSprSpeed().
// Routes to the appropriate getter based on actID category.
func (t *SprTable) GetSprSpeed(sprID, actID int) int {
	switch {
	case walkActions[actID]:
		return t.GetMoveSpeed(sprID, actID)
	case actID == ActSkillAttack:
		return t.GetDirSpellSpeed(sprID)
	case actID == ActSkillBuff:
		return t.GetNodirSpellSpeed(sprID)
	case attackActions[actID]:
		return t.GetAttackSpeed(sprID, actID)
	case actID == ActThink || actID == ActAggress:
		return t.GetSpecialSpeed(sprID, actID)
	default:
		return 0
	}
}
