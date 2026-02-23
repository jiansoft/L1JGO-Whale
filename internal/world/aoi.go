package world

// AOIGrid implements a cell-based Area of Interest system.
// Cell size is chosen so that a 3x3 neighbourhood of cells fully covers
// the visibility range (Chebyshev distance 20).
// Accessed only from the game loop goroutine — no locks.

const cellSize = 20

type cellKey struct {
	mapID int16
	cx    int32
	cy    int32
}

func toCellCoord(v int32) int32 {
	if v < 0 {
		return (v - cellSize + 1) / cellSize
	}
	return v / cellSize
}

// AOIGrid tracks which sessions are in which cells.
type AOIGrid struct {
	cells map[cellKey]map[uint64]struct{} // cellKey → set of sessionIDs
}

func NewAOIGrid() *AOIGrid {
	return &AOIGrid{
		cells: make(map[cellKey]map[uint64]struct{}),
	}
}

func (g *AOIGrid) key(x, y int32, mapID int16) cellKey {
	return cellKey{mapID: mapID, cx: toCellCoord(x), cy: toCellCoord(y)}
}

// Add places a session into the grid.
func (g *AOIGrid) Add(sessionID uint64, x, y int32, mapID int16) {
	k := g.key(x, y, mapID)
	cell := g.cells[k]
	if cell == nil {
		cell = make(map[uint64]struct{})
		g.cells[k] = cell
	}
	cell[sessionID] = struct{}{}
}

// Remove takes a session out of the grid.
func (g *AOIGrid) Remove(sessionID uint64, x, y int32, mapID int16) {
	k := g.key(x, y, mapID)
	cell := g.cells[k]
	if cell != nil {
		delete(cell, sessionID)
		if len(cell) == 0 {
			delete(g.cells, k)
		}
	}
}

// Move updates a session's cell when its position changes.
func (g *AOIGrid) Move(sessionID uint64, oldX, oldY int32, oldMap int16, newX, newY int32, newMap int16) {
	oldK := g.key(oldX, oldY, oldMap)
	newK := g.key(newX, newY, newMap)
	if oldK == newK {
		return
	}
	g.Remove(sessionID, oldX, oldY, oldMap)
	g.Add(sessionID, newX, newY, newMap)
}

// GetNearby returns all session IDs in a 3x3 neighbourhood of cells
// around the given position. Caller does fine-grained distance filtering.
func (g *AOIGrid) GetNearby(x, y int32, mapID int16) []uint64 {
	cx := toCellCoord(x)
	cy := toCellCoord(y)
	var result []uint64
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			k := cellKey{mapID: mapID, cx: cx + dx, cy: cy + dy}
			for sid := range g.cells[k] {
				result = append(result, sid)
			}
		}
	}
	return result
}

// NpcAOIGrid tracks which NPCs are in which cells.
// Same logic as AOIGrid but keyed by int32 NPC object IDs instead of uint64 session IDs.
// Separate type to avoid type assertions on the hot path.
type NpcAOIGrid struct {
	cells map[cellKey]map[int32]struct{}
}

func NewNpcAOIGrid() *NpcAOIGrid {
	return &NpcAOIGrid{
		cells: make(map[cellKey]map[int32]struct{}),
	}
}

func (g *NpcAOIGrid) key(x, y int32, mapID int16) cellKey {
	return cellKey{mapID: mapID, cx: toCellCoord(x), cy: toCellCoord(y)}
}

// Add places an NPC into the grid.
func (g *NpcAOIGrid) Add(npcID int32, x, y int32, mapID int16) {
	k := g.key(x, y, mapID)
	cell := g.cells[k]
	if cell == nil {
		cell = make(map[int32]struct{})
		g.cells[k] = cell
	}
	cell[npcID] = struct{}{}
}

// Remove takes an NPC out of the grid.
func (g *NpcAOIGrid) Remove(npcID int32, x, y int32, mapID int16) {
	k := g.key(x, y, mapID)
	cell := g.cells[k]
	if cell != nil {
		delete(cell, npcID)
		if len(cell) == 0 {
			delete(g.cells, k)
		}
	}
}

// Move updates an NPC's cell when its position changes.
func (g *NpcAOIGrid) Move(npcID int32, oldX, oldY int32, oldMap int16, newX, newY int32, newMap int16) {
	oldK := g.key(oldX, oldY, oldMap)
	newK := g.key(newX, newY, newMap)
	if oldK == newK {
		return
	}
	g.Remove(npcID, oldX, oldY, oldMap)
	g.Add(npcID, newX, newY, newMap)
}

// GetNearby returns all NPC IDs in a 3x3 neighbourhood of cells.
func (g *NpcAOIGrid) GetNearby(x, y int32, mapID int16) []int32 {
	cx := toCellCoord(x)
	cy := toCellCoord(y)
	var result []int32
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			k := cellKey{mapID: mapID, cx: cx + dx, cy: cy + dy}
			for nid := range g.cells[k] {
				result = append(result, nid)
			}
		}
	}
	return result
}
