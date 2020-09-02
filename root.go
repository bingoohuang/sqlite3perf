package sqlite3perf

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"hash"
	"log"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// nolint:gochecknoglobals
var (
	cfgFile string
	dbPath  string

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
	if err := rootCmd.Execute(); err != nil {
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

func setupBench(clear bool) *sql.DB {
	log.Println("Opening database")
	// Database Setup
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error while opening database '%s': %s", dbPath, err.Error())
	}

	log.Println("Dropping table 'bench' if already present")

	if clear {
		if _, err := db.Exec("DROP TABLE IF EXISTS bench"); err != nil {
			log.Fatalf("Could not delete table 'bench' for (re-)generation of data: %s", err)
		}

		log.Println("(Re-)creating table 'bench'")

		if _, err := db.Exec("CREATE TABLE bench(ID int PRIMARY KEY ASC, rand TEXT, hash TEXT)"); err != nil {
			log.Fatalf("Could not create table 'bench': %s", err)
		}

	}
	log.Println("Setting up the environment")
	return db
}

type Hash struct {
	b []byte
	h hash.Hash
}

func NewHash() *Hash {
	return &Hash{
		// We use a 8 byte random value as this is the optimal size for SHA256, which operates on 64bit blocks
		b: make([]byte, 8),
		// Initialize the hasher once and reuse it using Reset()
		h: sha256.New(),
	}
}

func (h *Hash) Gen() (randstr, hash string) {
	if _, err := rand.Read(h.b); err != nil {
		log.Fatalf("Can not read random values: %s", err)
	}

	h.h.Reset()           // Reset the hasher so we can reuse it
	_, _ = h.h.Write(h.b) // Fill the hasher

	return hex.EncodeToString(h.b), hex.EncodeToString(h.h.Sum(nil))
}
