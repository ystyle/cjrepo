package migrations

import (
	"fmt"
	"log"
	"sort"

	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

type MigrationFunc func(engine *xorm.Engine) error

type Migration struct {
	Version string
	Name    string
	Execute MigrationFunc
}

var registeredMigrations []Migration

func Register(m Migration) {
	registeredMigrations = append(registeredMigrations, m)
}

func GetRegistered() []Migration {
	return registeredMigrations
}

func Run(engine *xorm.Engine) error {
	if err := engine.Sync2(new(models.Migration)); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	executed := make(map[string]bool)
	var executedMigrations []models.Migration
	if err := engine.Find(&executedMigrations); err != nil {
		return fmt.Errorf("failed to query executed migrations: %w", err)
	}
	for _, m := range executedMigrations {
		executed[m.Version] = true
	}

	sort.Slice(registeredMigrations, func(i, j int) bool {
		return registeredMigrations[i].Version < registeredMigrations[j].Version
	})

	for _, m := range registeredMigrations {
		if executed[m.Version] {
			log.Printf("[Migration] %s already executed, skipping", m.Version)
			continue
		}

		log.Printf("[Migration] Executing %s: %s", m.Version, m.Name)

		if err := m.Execute(engine); err != nil {
			return fmt.Errorf("migration %s failed: %w", m.Version, err)
		}

		record := &models.Migration{
			Version: m.Version,
			Name:    m.Name,
		}
		if _, err := engine.Insert(record); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", m.Version, err)
		}

		log.Printf("[Migration] %s completed successfully", m.Version)
	}

	return nil
}

func IsExecuted(engine *xorm.Engine, version string) (bool, error) {
	return engine.Where("version = ?", version).Exist(&models.Migration{})
}