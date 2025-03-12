package model

import (
	"fmt"

	"github.com/jimmywmt/twstockchip/config"
	gorm_logrus "github.com/onrik/gorm-logrus"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	SqliteDB   *gorm.DB
	PostgresDB *gorm.DB
)

// BuildPostgresDSN generates a PostgreSQL DSN (Data Source Name) string from config.Config
func BuildPostgresDSN(cfg *config.PostgreConfig) string {
	// Set SSL mode based on UseSSL field
	sslMode := "disable"
	if cfg.UseSSL {
		sslMode = "require"
	}

	// Build the DSN string
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, sslMode,
	)

	return dsn
}

func InitDBModel(sqliteFile string, postgresDSN string) {
	var err error

	if sqliteFile != "" {
		// Initialize SQLite
		SqliteDB, err = gorm.Open(sqlite.Open(sqliteFile), &gorm.Config{
			Logger: gorm_logrus.New(),
		})
		if err != nil {
			log.WithError(err).Fatalln("連結SQLite資料庫失敗")
		}
		SqliteDB.AutoMigrate(&Stock{}, &Dealer{}, &Transaction{})
	}

	if postgresDSN != "" {
		// Initialize PostgreSQL
		PostgresDB, err = gorm.Open(postgres.Open(postgresDSN), &gorm.Config{
			Logger: gorm_logrus.New(),
		})
		if err != nil {
			log.WithError(err).Fatalln("連結PostgreSQL資料庫失敗")
		}
		PostgresDB.AutoMigrate(&Stock{}, &Dealer{}, &Transaction{})
	}
}
