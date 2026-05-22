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
	db.Exec("CREATE DATABASE IF NOT EXISTS future CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	db.Exec("USE future")
	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		email VARCHAR(100) UNIQUE,
		phone VARCHAR(20),
		nickname VARCHAR(50),
		avatar VARCHAR(255),
		role VARCHAR(20) DEFAULT 'user',
		status VARCHAR(20) DEFAULT 'active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		last_login_at TIMESTAMP NULL,
		login_count INT DEFAULT 0,
		last_ip VARCHAR(50),
		email_verified TINYINT(1) DEFAULT 0,
		phone_verified TINYINT(1) DEFAULT 0,
		bio TEXT,
		location VARCHAR(100),
		birthday DATE,
		gender VARCHAR(10),
		language VARCHAR(10) DEFAULT 'zh-CN',
		timezone VARCHAR(50) DEFAULT 'Asia/Shanghai'
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