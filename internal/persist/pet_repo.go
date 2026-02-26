package persist

import (
	"context"
)

// PetRow represents a persisted pet record keyed by amulet item object ID.
type PetRow struct {
	ItemObjID int32  // Amulet item ObjectID (primary key)
	ObjID     int32  // Pet NPC object ID when spawned
	NpcID     int32  // Current NPC template ID (changes on evolution)
	Name      string // Pet display name
	Level     int16
	HP        int32
	MaxHP     int32
	MP        int32
	MaxMP     int32
	Exp       int32
	Lawful    int32
}

// PetRepo handles CRUD operations for the character_pets table.
type PetRepo struct {
	db *DB
}

// NewPetRepo creates a new PetRepo.
func NewPetRepo(db *DB) *PetRepo {
	return &PetRepo{db: db}
}

// LoadByItemObjID loads a single pet by its amulet item object ID.
func (r *PetRepo) LoadByItemObjID(ctx context.Context, itemObjID int32) (*PetRow, error) {
	row := r.db.Pool.QueryRow(ctx,
		`SELECT item_obj_id, obj_id, npc_id, name, level, hp, hpmax, mp, mpmax, exp, lawful
		 FROM character_pets WHERE item_obj_id = $1`, itemObjID,
	)
	var p PetRow
	if err := row.Scan(
		&p.ItemObjID, &p.ObjID, &p.NpcID, &p.Name,
		&p.Level, &p.HP, &p.MaxHP, &p.MP, &p.MaxMP, &p.Exp, &p.Lawful,
	); err != nil {
		return nil, err
	}
	return &p, nil
}

// Save inserts or updates a pet record (upsert by item_obj_id).
func (r *PetRepo) Save(ctx context.Context, p *PetRow) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO character_pets (item_obj_id, obj_id, npc_id, name, level, hp, hpmax, mp, mpmax, exp, lawful)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (item_obj_id) DO UPDATE SET
		   obj_id = EXCLUDED.obj_id,
		   npc_id = EXCLUDED.npc_id,
		   name   = EXCLUDED.name,
		   level  = EXCLUDED.level,
		   hp     = EXCLUDED.hp,
		   hpmax  = EXCLUDED.hpmax,
		   mp     = EXCLUDED.mp,
		   mpmax  = EXCLUDED.mpmax,
		   exp    = EXCLUDED.exp,
		   lawful = EXCLUDED.lawful`,
		p.ItemObjID, p.ObjID, p.NpcID, p.Name,
		p.Level, p.HP, p.MaxHP, p.MP, p.MaxMP, p.Exp, p.Lawful,
	)
	return err
}

// Delete removes a pet record by amulet item object ID.
func (r *PetRepo) Delete(ctx context.Context, itemObjID int32) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM character_pets WHERE item_obj_id = $1`, itemObjID,
	)
	return err
}
