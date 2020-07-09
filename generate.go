package sqlite3perf

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // import sqlite3 driver
	"github.com/spf13/cobra"
)

// nolint:gochecknoglobals
var (
	numRecs    int
	batchSize  int
	vacuum     bool
	logSeconds int

	// generateCmd represents the generate command
	generateCmd = &cobra.Command{
		Use:   "generate",
		Short: "generate records to benchmark against",
		Long: `This command generates records to benchmark against.
Each record consists of an ID, a 8 byte hex encoded random value
and a SHA256 hash of said random value.

ATTENTION: The 'bench' table will be DROPPED each time this command is called, before it
is (re)-generated!
	`,
		Run: generateRun,
	}
)

func generateRun(cmd *cobra.Command, args []string) {
	log.Printf("Generating %d records", numRecs)

	log.Println("Opening database")
	// Database Setup
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error while opening database '%s': %s", dbPath, err.Error())
	}
	defer db.Close()

	log.Println("Dropping table 'bench' if already present")

	if _, err := db.Exec("DROP TABLE IF EXISTS bench"); err != nil {
		log.Fatalf("Could not delete table 'bench' for (re-)generation of data: %s", err)
	}

	log.Println("(Re-)creating table 'bench'")

	if _, err := db.Exec("CREATE TABLE bench (ID int PRIMARY KEY ASC, rand TEXT, hash TEXT);"); err != nil {
		log.Fatalf("Could not create table 'bench': %s", err)
	}

	log.Println("Setting up the environment")

	// Preinitialize i so that we can use it in a goroutine to give proper feedback
	var i int
	// Set up logging mechanism. We use a goroutine here which logs the
	// records already generated every two seconds until "done" is signaled
	// via the channel.
	done := make(chan bool)
	start := time.Now()

	go inserts(&i, db, done)

	progressLogging(start, &i, done)

	if vacuum {
		vacuumDB(db)
	}
}

// nolint:gomnd,gosec
func inserts(i *int, db *sql.DB, done chan bool) {
	// We use a 8 byte random value as this is the optimal size for SHA256,
	// which operates on 64bit blocks
	b := make([]byte, 8)
	// Initialize the hasher once and reuse it using Reset()
	h := sha256.New()
	// Prepare values needed so that there aren't any allocations done in the loop
	query := "INSERT INTO bench(ID, rand, hash) VALUES" +
		strings.Repeat(",(?,?,?)", batchSize)[1:]

	ps, _ := db.Prepare(query)
	defer ps.Close()

	lastNum := numRecs % batchSize

	// Start generation of actual records
	log.Println("Starting inserts")

	args := make([]interface{}, 0, batchSize*3)

	for *i = 0; *i < numRecs; *i++ {
		if _, err := rand.Read(b); err != nil {
			log.Fatalf("Can not read random values: %s", err)
		}

		h.Reset()         // Reset the hasher so we can reuse it
		_, _ = h.Write(b) // Fill the hasher

		hexB := hex.EncodeToString(b)
		hashB := hex.EncodeToString(h.Sum(nil))

		args = append(args, *i, hexB, hashB)

		if len(args) == batchSize*3 {
			if _, err := ps.Exec(args...); err != nil {
				log.Fatalf("Inserting values into database failed: %s", err)
			}

			args = args[0:0]
		} else if lastNum > 0 && *i+1 == numRecs {
			query := "INSERT INTO bench(ID, rand, hash) VALUES" +
				strings.Repeat(",(?,?,?)", lastNum)[1:]
			if _, err := db.Exec(query, args...); err != nil {
				log.Fatalf("Inserting values into database failed: %s", err)
			}
		}
	}

	// Signal the progress log that we are done
	done <- true
}

// nolint:gomnd
func progressLogging(start time.Time, i *int, done chan bool) {
	log.Println("Starting progress logging")

	l := len(fmt.Sprintf("%d", numRecs))
	// Precalculate the percentage each record represents
	p := float64(100) / float64(numRecs)

	ticker := time.NewTicker(time.Duration(logSeconds) * time.Second)
	defer ticker.Stop()

out:
	for {
		select {
		// Since this is a time consuming process depending on the number of
		// records	created, we want some feedback every 2 seconds
		case <-ticker.C:
			dur := time.Since(start)
			log.Printf("%*d/%*d (%6.2f%%) written in %s, avg: %s/record, %2.2f records/s",
				l, *i, l, numRecs, p*float64(*i), dur,
				time.Duration(dur.Nanoseconds()/int64(*i)), float64(*i)/dur.Seconds())
		case <-done:
			break out
		}
	}

	dur := time.Since(start)
	log.Printf("%*d/%*d (%6.2f%%) written in %s, avg: %s/record, %2.2f records/s",
		l, numRecs, l, numRecs, p*float64(numRecs), dur,
		time.Duration(dur.Nanoseconds()/int64(numRecs)), float64(numRecs)/dur.Seconds())
}

func vacuumDB(db *sql.DB) {
	log.Println("Vaccumating database file")

	start := time.Now()

	if _, err := db.Exec("VACUUM"); err != nil {
		log.Printf("Vacuumating database caused an error: %s", err)
		log.Println("Proceed with according caution.")
	}

	since := time.Since(start)
	log.Printf("Vacuumation took %s", since)
}

// nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(generateCmd)

	// Here you will define your flags and configuration settings.
	flagSet := generateCmd.Flags()
	flagSet.IntVarP(&numRecs, "records", "r", 1000,
		"number of records to generate")
	flagSet.IntVarP(&batchSize, "batch", "b", 100,
		"number of records as a batch to insert at one time")
	flagSet.IntVarP(&logSeconds, "interval", "i", 2,
		"interval between progress messages")
	flagSet.BoolVarP(&vacuum, "vacuum", "v", false,
		"VACUUM database file after the records were generated.")
}
