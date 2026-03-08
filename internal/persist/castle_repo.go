package persist

import (
	"context"
	"time"
)

// CastleState 城堡動態狀態（從 DB 載入）。
type CastleState struct {
	CastleID    int32
	CastleName  string
	TaxRate     int32
	PublicMoney int64
	WarTime     time.Time
}

// CastleRepo 城堡持久化操作。
type CastleRepo struct {
	db *DB
}

// NewCastleRepo 建構城堡 repo。
func NewCastleRepo(db *DB) *CastleRepo {
	return &CastleRepo{db: db}
}

// LoadAll 載入所有城堡動態狀態。
func (r *CastleRepo) LoadAll(ctx context.Context) ([]CastleState, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT castle_id, castle_name, tax_rate, public_money, war_time
		 FROM castles ORDER BY castle_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CastleState
	for rows.Next() {
		var c CastleState
		if err := rows.Scan(&c.CastleID, &c.CastleName, &c.TaxRate, &c.PublicMoney, &c.WarTime); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// UpdateTaxRate 更新城堡稅率。
func (r *CastleRepo) UpdateTaxRate(ctx context.Context, castleID int32, rate int32) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE castles SET tax_rate = $1 WHERE castle_id = $2`,
		rate, castleID)
	return err
}

// UpdatePublicMoney 更新城堡寶庫金額。
func (r *CastleRepo) UpdatePublicMoney(ctx context.Context, castleID int32, money int64) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE castles SET public_money = $1 WHERE castle_id = $2`,
		money, castleID)
	return err
}

// UpdateWarTime 更新攻城戰時間。
func (r *CastleRepo) UpdateWarTime(ctx context.Context, castleID int32, warTime time.Time) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE castles SET war_time = $1 WHERE castle_id = $2`,
		warTime, castleID)
	return err
}

// ResetForWar 攻城戰開始時重置稅率和寶庫（Java: taxRate=10, publicMoney=0）。
func (r *CastleRepo) ResetForWar(ctx context.Context, castleID int32) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE castles SET tax_rate = 10, public_money = 0 WHERE castle_id = $1`,
		castleID)
	return err
}
