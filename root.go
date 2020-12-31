package sqlite3perf

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"hash"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/m1ome/randstr"
	"github.com/valyala/fastrand"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// nolint:gochecknoglobals
var (
	cfgFile    string
	driverName string
	dbPath     string
	table      string

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "sqlite3perf",
		Short: "small application to judge Golang's performance with SQLite3",
		Long: `This application was built while researching the answer to
the question https://stackoverflow.com/questions/48000940/.

It consists of two parts: the go binary you just called and 'bench.py',
which is as much of a Python implementation of the 'bench' command as I am
capable of (improvements more than welcome!).

You first have to fill the database with the 'generate' command, after which you
can call the 'bench' command to see how Go performs with SQLite3.

After the database is filled, one can run 'bench.py' against it
	`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		//	Run: func(cmd *cobra.Command, args []string) { },
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	ctx, cancel := context.WithCancel(context.Background())
	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Interrupt)

	go func() {
		<-kill // trap Ctrl+C and call cancel on the context
		cancel()
	}()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// nolint:gochecknoinits,wsl
func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&cfgFile, "config", "c", "",
		"config file (default is $HOME/.sqlite3perf.yaml)")
	pf.StringVar(&driverName, "driverName", "sqlite3", "driver name(sqlite3/mysql)")
	pf.StringVar(&table, "table", "bench", "table name(bench/ff)")
	pf.StringVar(&dbPath, "db",
		"./sqlite3perf.db?_journal=wal&_sync=0", "path to database")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".sqlite3perf" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".sqlite3perf")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// Table defines the structure of perference table information.
type Table struct {
	InsertFieldsNum int
	DropSQL         string
	CreateSQL       string
	Generator       func(i int) []interface{}
	CreateInsertSQL func(batchSize int) string
}

var tables = map[string]Table{
	"bench": {
		DropSQL:         `DROP TABLE IF EXISTS bench`,
		CreateSQL:       `CREATE TABLE bench(ID int PRIMARY KEY, rand varchar(100), hash varchar(100))`,
		InsertFieldsNum: 3,
		Generator:       NewHasher().Generator,
		CreateInsertSQL: func(batchSize int) string {
			return "INSERT INTO bench(ID, rand, hash) VALUES" + strings.Repeat(",(?,?,?)", batchSize)[1:]
		},
	},

	"ff": {
		DropSQL: `DROP TABLE IF EXISTS ff`,
		CreateSQL: `CREATE TABLE ff (
		  id bigint(20) NOT NULL AUTO_INCREMENT,
		  f01 varchar(255),
		  f02 varchar(255),
		  f03 varchar(255),
		  f04 varchar(255),
		  f05 varchar(255),
		  f06 varchar(255),
		  f07 varchar(255),
		  f08 varchar(255),
		  f09 varchar(255),
		  f10 varchar(255),
		  f11 varchar(255),
		  f12 varchar(255),
		  f13 varchar(255),
		  f14 varchar(255),
		  f15 varchar(255),
		  f16 varchar(255),
		  f17 varchar(255),
		  f18 varchar(255),
		  created datetime NOT NULL COMMENT '创建时间',
		  updated datetime NOT NULL COMMENT '更新时间',
		  PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT = '测试批量插入表'`,
		InsertFieldsNum: 20,
		Generator: func(i int) []interface{} {
			vars := make([]interface{}, 20)

			for i := 0; i < 18; i++ {
				vars[i] = randstr.GetString(int(fastrand.Uint32n(250) + 5))
			}

			vars[18], vars[19] = time.Now(), time.Now()

			return vars
		},
		CreateInsertSQL: func(batchSize int) string {
			return "INSERT INTO ff(f01, f02, f03, f04, f05, f06, f07, f08, f09, f10, " +
				"f11, f12, f13, f14, f15, f16, f17, f18, created, updated) VALUES" +
				strings.Repeat(",(?,?,?,?,?, ?,?,?,?,?, ?,?,?,?,?, ?,?,?,?,?)", batchSize)[1:]
		},
	},
}

func setupBench(clear bool, maxOpenConns int) *sql.DB {
	log.Print("Opening database")
	db, err := sql.Open(driverName, dbPath)
	if err != nil {
		log.Fatalf("Error while opening database '%s': %s", dbPath, err.Error())
	}

	if maxOpenConns > 0 {
		db.SetMaxOpenConns(maxOpenConns)
	}

	if clear {
		log.Print("Dropping table", table, "if already present")

		t, ok := tables[table]
		if !ok {
			log.Fatalf("%s does not exist", table)
		}

		if _, err := db.Exec(t.DropSQL); err != nil {
			log.Fatalf("Could not delete table 'bench' for (re-)generation of data: %s", err)
		}

		log.Print("(Re-)creating table", table)

		if _, err := db.Exec(t.CreateSQL); err != nil {
			log.Fatalf("Could not create table %s: %s", table, err)
		}

		log.Print("Setting up the environment")
	}

	return db
}

// Hasher is a structure to generate a random string with its hash value.
type Hasher struct {
	b []byte
	h hash.Hash
}

// NewHasher creates a new Hasher instance.
func NewHasher() *Hasher {
	return &Hasher{
		// We use a 8 byte random value as this is the optimal size for SHA256, which operates on 64bit blocks
		b: make([]byte, 8),
		// Initialize the hasher once and reuse it using Reset()
		h: sha256.New(),
	}
}

// Generator generates a random string and its hash value.
func (h *Hasher) Generator(index int) []interface{} {
	ret := make([]interface{}, 3)
	ret[0] = index
	ret[1], ret[2] = h.Gen()
	return ret
}

// Gen generates a random string and its hash value.
func (h *Hasher) Gen() (randstr, hash string) {
	if _, err := rand.Read(h.b); err != nil {
		log.Fatalf("Can not read random values: %s", err)
	}

	h.h.Reset()           // Reset the hasher so we can reuse it
	_, _ = h.h.Write(h.b) // Fill the hasher

	return hex.EncodeToString(h.b), hex.EncodeToString(h.h.Sum(nil))
}

// SleepContext sleep within a context.
func SleepContext(ctx context.Context, delay time.Duration) bool {
	timeout, timeoutFn := context.WithTimeout(ctx, delay)
	defer timeoutFn()
	<-timeout.Done()
	return timeout.Err() != nil
}
