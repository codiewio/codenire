package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"
)

var dbCountMetric = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "sand_pg_database_count",
	Help: "Number of databases in PostgreSQL",
})

func createDB(dbName, userName, userPassword string) error {
	dsn := *isolatedPostgresDSN

	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return fmt.Errorf("error connecting to the database: %w", err)
	}
	defer func() {
		_ = conn.Close(context.Background())
	}()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	createDBSQL := fmt.Sprintf("CREATE DATABASE %s;", dbName)
	_, err = conn.Exec(timeoutCtx, createDBSQL)
	if err != nil {
		return fmt.Errorf("error creating the database: %w", err)
	}

	createUserSQL := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s';", userName, userPassword)
	_, err = conn.Exec(timeoutCtx, createUserSQL)
	if err != nil {
		return fmt.Errorf("error creating the user: %w", err)
	}

	// in new db create access

	newDSN := strings.Replace(dsn, "/postgres", "/"+dbName, 1)

	newConn, err := pgx.Connect(context.Background(), newDSN)
	if err != nil {
		return fmt.Errorf("error connecting to new database: %w", err)
	}
	defer newConn.Close(context.Background())

	alterDBSQL := fmt.Sprintf("ALTER DATABASE %s OWNER TO %s;", dbName, userName)
	_, err = newConn.Exec(timeoutCtx, alterDBSQL)
	if err != nil {
		return fmt.Errorf("error changing database owner: %w", err)
	}

	grantSchemaSQL := fmt.Sprintf("ALTER SCHEMA public OWNER TO %s;", userName)
	_, err = newConn.Exec(timeoutCtx, grantSchemaSQL)
	if err != nil {
		return fmt.Errorf("error granting schema ownership: %w", err)
	}

	grantUsageSQL := fmt.Sprintf("GRANT ALL ON SCHEMA public TO %s;", userName)
	_, err = newConn.Exec(timeoutCtx, grantUsageSQL)
	if err != nil {
		return fmt.Errorf("error granting schema privileges: %w", err)
	}

	return nil
}

func dropDB(dbName string) error {
	dsn := *isolatedPostgresDSN

	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return fmt.Errorf("error connecting to the database: %w", err)
	}
	defer func() {
		_ = conn.Close(context.Background())
	}()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	terminateConnsSQL := fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s' AND pid <> pg_backend_pid();
	`, dbName)

	_, err = conn.Exec(timeoutCtx, terminateConnsSQL)
	if err != nil {
		return fmt.Errorf("warning: failed to terminate active connections to database %s: %w", dbName, err)
	}

	_, err = conn.Exec(timeoutCtx, fmt.Sprintf("DROP DATABASE %s;", dbName))
	if err != nil {
		if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
			log.Printf("error: timeout exceeded while dropping database %s", dbName)
		}
		return fmt.Errorf("error dropping database %s: %w", dbName, err)
	}

	return nil
}

func updateDatabaseCountMetric() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dsn := *isolatedPostgresDSN

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Printf("Failed to connect to the database: %v", err)
		return
	}
	defer conn.Close(ctx)

	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM pg_database").Scan(&count)
	if err != nil {
		log.Printf("Error executing the query: %v", err)
		return
	}

	dbCountMetric.Set(float64(count))
}
