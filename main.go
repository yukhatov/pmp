package main

import (
	"math/rand"
	"net/http"
	"os/signal"
	"runtime"
	"syscall"

	"bitbucket.org/tapgerine/pmp/control"

	"bitbucket.org/tapgerine/pmp/control/config"
	"bitbucket.org/tapgerine/pmp/control/database"

	"time"

	"flag"

	"fmt"

	"log/syslog"

	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/Sirupsen/logrus/hooks/syslog"
	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/roistat/go-clickhouse"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	log.SetFormatter(&log.JSONFormatter{})
	//log.SetOutput(os.Stdout)

	hook, err := logrus_syslog.NewSyslogHook("", "", syslog.LOG_INFO, "")
	if err != nil {
		log.Error("Unable to connect to local syslog daemon")
	} else {
		log.AddHook(hook)
	}
	// Only log the warning severity or above.
	//log.SetLevel(log.InfoLevel)
}

// Main Program
func main() {
	var (
		dbName           = flag.String("db_name", "pmp", "Postgres database name")
		postgresUser     = flag.String("postgres_user", "pmp", "Postgres user name")
		postgresHost     = flag.String("postgres_host", "localhost", "Postgres host name")
		postgresPort     = flag.String("postgres_port", "5432", "Postgres port name")
		postgresPwd      = flag.String("postgres_pwd", "", "Postgres user password")
		redisHost        = flag.String("redis_host", "localhost", "Redis host")
		redisHost2       = flag.String("redis_host2", "localhost", "Redis host")
		redisPwd         = flag.String("redis_pwd", "", "Redis password")
		clickHouseClient = flag.String("clickhouse_client", "localhost:8123", "ClickHouse client host")
		uploadFolder     = flag.String("upload_folder", "/tmp/upload", "Upload folder path")
		rotatorDomain    = flag.String("rotator_domain", "pmp.tapgerine.com", "Rotator domain")
	)
	flag.Parse()

	var err error
	database.Postgres, err = gorm.Open(
		"postgres",
		fmt.Sprintf("host=%s port=%s dbname=%s sslmode=disable user=%s password=%s", *postgresHost, *postgresPort, *dbName, *postgresUser, *postgresPwd),
	)
	if err != nil {
		panic(err)
	}
	defer database.Postgres.Close()

	database.Redis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:6379", *redisHost),
		Password: *redisPwd, // no password set
		DB:       0,         // use default DB
	})
	defer database.Redis.Close()

	database.Redis2 = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:6379", *redisHost2),
		Password: *redisPwd, // no password set
		DB:       0,         // use default DB
	})
	defer database.Redis2.Close()

	database.ClickHouse = clickhouse.NewConn(*clickHouseClient, clickhouse.NewHttpTransport())
	err = database.ClickHouse.Ping()
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	database.FilesManager = &database.FilesManagerClient{RootFolder: *uploadFolder}

	config.RotatorDomain = *rotatorDomain

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill, syscall.SIGTERM)

	go func() {
		<-ch
		signal.Stop(ch)
		//os.Remove(PIDFile)
		log.Info("Stopping service")
		os.Exit(0)

	}()

	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Info("Application working on port 8080")
	log.Fatal(http.ListenAndServe(":8080", control.CreateRouter()))
}
