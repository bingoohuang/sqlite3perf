package sqlite3perf

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
)

// nolint:gochecknoinits,wsl
func init() {
	// benchCmd represents the bench command
	c := &cobra.Command{
		Use:   "bench",
		Short: "do a simple benchmark",
		Long: `A simple benchmark is done using the records created with the "generate" command.

	All records are retrieved and the saved random value is decoded from hex,
	hashed with SHA256 and compared with the hash saved to the database.
	"`,
		Run: benchRun,
	}

	rootCmd.AddCommand(c)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// benchCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// benchCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func benchRun(cmd *cobra.Command, args []string) {
	log.Print("Running benchmark")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("Error while opening database '%s': %s", dbPath, err.Error())
		os.Exit(1)
	}
	defer db.Close()

	var (
		ID   int64
		rand string
		hash string
	)

	start := time.Now()

	rows, err := db.Query("select * from bench")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	log.Printf("Time after query: %s", time.Since(start))
	log.Printf("Beginning loop")

	h := sha256.New()

	done := make(chan bool)
	c := int64(0)
	l := time.Now()

	go func() {
		t := time.NewTicker(time.Second * 2).C
		for {
			select {
			case <-t:
				log.Printf("%d rows processed", atomic.LoadInt64(&c))
			case <-done:
				return
			}
		}
	}()

	pre := time.Now()
	for ; rows.Next(); atomic.AddInt64(&c, 1) {
		if err = rows.Scan(&ID, &rand, &hash); err != nil {
			log.Fatal(err)
		}

		if atomic.LoadInt64(&c) == 0 {
			log.Printf("Acessing the first result set \n\tID %d,\n\trand: %s,\n\thash: %s\ntook %s",
				ID, rand, hash, time.Since(pre))
		}

		h.Reset()

		// Do something halfway useful
		v, err := hex.DecodeString(rand)
		if err != nil {
			log.Print("Could not decode hex string of original value to byte slice: ", err)
			// Not necessarily fatal, so we can...
			continue
		}

		_, _ = h.Write(v)
		if hex.EncodeToString(h.Sum(nil)) != hash {
			log.Fatal("Hash of original value and persist hash do not match!")
		}
	}
	e := time.Since(start)
	el := time.Since(l)
	done <- true

	totalRows := atomic.LoadInt64(&c)
	log.Printf("%d rows processed", totalRows)
	log.Printf("Finished loop after %s", el)

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	if totalRows > 0 {
		duration := time.Duration(e.Nanoseconds() / totalRows)
		log.Printf("Average %s per record, %s overall", duration, e)
	}
}
