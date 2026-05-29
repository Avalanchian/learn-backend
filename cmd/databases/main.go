// dbping.go
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5"
)

func main() {
	timeout := flag.Duration("timeout", 5*time.Second, "timeout for postgres connection")
	flag.Parse()

	cfg, err := pgConfigFromEnv()
	if err != nil {
		log.Fatalf("postgres config error: %v", err)
	}

	// setup embeded postgres server
	portN, err := strconv.Atoi(cfg.port)
	if err != nil {
		panic(err)
	}

	embeddedCfg := embeddedpostgres.DefaultConfig().
		Username(cfg.user).
		Password(cfg.password).
		Database(cfg.database).
		Port(uint32(portN)).
		Logger(io.Discard)

	embeddedDB := embeddedpostgres.NewDatabase(embeddedCfg)
	if err := embeddedDB.Start(); err != nil {
		panic(err)
	}

	log.Printf("postgres is running on %s\n", embeddedCfg.GetConnectionURL())
	defer embeddedDB.Stop()

	db, err := sql.Open("postgres", cfg.String())
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		panic(err)
	}
	log.Println("ping successful")
}

type pgConfig struct {
	user, password, database, host, port string //required
	sslMode                              string //optional
}

func pgConfigFromEnv() (pgConfig, error) {
	var missing []string

	// closure to avoid code duplication
	get := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			missing = append(missing, key)
		}
		return val
	}

	cfg := pgConfig{
		user:     get("PG_USER"),
		password: get("PG_PASSWORD"),
		database: get("PG_DATABASE"),
		host:     get("PG_HOST"),
		port:     get("PG_PORT"),
		sslMode:  os.Getenv("PG_SSLMODE"),
	}

	switch cfg.sslMode {
	case "", "disable", "allow", "require", "verify-ca", "verify-full":
		//valid ssl mode
	default:
		return cfg, fmt.Errorf(`invalid sslmode "%s": expected one of "", "disable", "allow", "require", "verify-ca", "verify-full"`, cfg.sslMode)
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return cfg, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}

func (pg pgConfig) String() string {
	s := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", pg.user, pg.password, pg.host, pg.port, pg.database)
	if pg.sslMode != "" {
		s += "?sslmode=" + pg.sslMode
	}
	return s
}
