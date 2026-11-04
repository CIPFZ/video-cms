package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CIPFZ/video-cms/internal/core/config"
	"go.uber.org/zap"
)

func TestInitDatabaseRejectsUnsupportedDriver(t *testing.T) {
	_, err := InitDatabase(config.Database{Driver: "oracle"}, zap.NewNop())
	if err == nil {
		t.Fatal("InitDatabase() error = nil, want unsupported driver error")
	}
}

func TestNormalizeDriverAliases(t *testing.T) {
	tests := map[string]string{
		"":           "mysql",
		"mysql":      "mysql",
		"postgres":   "postgres",
		"postgresql": "postgres",
		"pgsql":      "postgres",
		"sqlite3":    "sqlite3",
	}
	for in, want := range tests {
		if got := normalizeDriver(in); got != want {
			t.Fatalf("normalizeDriver(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInitDatabaseWithSQLite3(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data", "app.db")

	gormDB, err := InitDatabase(config.Database{
		Driver: "sqlite3",
		SQLite: config.SQLite{
			Path:          dbPath,
			WAL:           true,
			BusyTimeoutMS: 3000,
			ForeignKeys:   true,
			MaxIdleConns:  1,
			MaxOpenConns:  1,
			LogMode:       "warn",
		},
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("InitDatabase() error = %v", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestInitDatabaseWithSQLite3CreatesDatabaseDirectory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "sqlite", "app.db")

	gormDB, err := InitDatabase(config.Database{
		Driver: "sqlite3",
		SQLite: config.SQLite{
			Path:         dbPath,
			MaxIdleConns: 1,
			MaxOpenConns: 1,
		},
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("InitDatabase() error = %v", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("sqlite db path stat error = %v, want file to exist", err)
	}
}

func TestInitDatabaseWithSQLite3AppliesPragmas(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data", "app.db")

	gormDB, err := InitDatabase(config.Database{
		Driver: "sqlite3",
		SQLite: config.SQLite{
			Path:          dbPath,
			WAL:           true,
			BusyTimeoutMS: 4321,
			ForeignKeys:   true,
			MaxIdleConns:  1,
			MaxOpenConns:  1,
		},
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("InitDatabase() error = %v", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	var journalMode string
	if err := gormDB.Raw("PRAGMA journal_mode").Scan(&journalMode).Error; err != nil {
		t.Fatalf("journal_mode query error = %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	var busyTimeout int
	if err := gormDB.Raw("PRAGMA busy_timeout").Scan(&busyTimeout).Error; err != nil {
		t.Fatalf("busy_timeout query error = %v", err)
	}
	if busyTimeout != 4321 {
		t.Fatalf("busy_timeout = %d, want 4321", busyTimeout)
	}

	var foreignKeys int
	if err := gormDB.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error; err != nil {
		t.Fatalf("foreign_keys query error = %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}
}
