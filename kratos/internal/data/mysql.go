package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-kratos/kratos/v2/log"
)

type Data struct {
	db *sql.DB
}

func NewData(logger log.Logger) (*Data, func(), error) {
	dsn := "root:mysql_2SASaZ@tcp(115.190.54.31:3306)/?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open mysql: %w", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err = db.Ping(); err != nil {
		return nil, nil, fmt.Errorf("mysql ping failed: %w", err)
	}

	// 自动建库建表
	initSchema(db)
	log.NewHelper(logger).Info("MySQL connected, schema ready")

	cleanup := func() {
		db.Close()
	}

	return &Data{db: db}, cleanup, nil
}

func initSchema(db *sql.DB) {
	db.Exec("CREATE DATABASE IF NOT EXISTS login_app CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	db.Exec("USE login_app")
	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
}

func (d *Data) DB() *sql.DB {
	return d.db
}

func (d *Data) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

func (d *Data) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}