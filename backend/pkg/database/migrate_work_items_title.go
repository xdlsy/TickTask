package database

import "gorm.io/gorm"

// MigrateWorkItemsTitleBackfill 把 source='manual' 且 title 为空但 activity 非空的 items 的 title 回填为 activity。
// 幂等：WHERE 条件保证重复调用不会覆盖已填好的 title。
func MigrateWorkItemsTitleBackfill(db *gorm.DB) error {
	return db.Exec(`
		UPDATE work_items
		SET title = activity
		WHERE source = 'manual'
		  AND (title = '' OR title IS NULL)
		  AND activity IS NOT NULL
	`).Error
}
