package sqlite3perf

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"go.uber.org/atomic"

	"github.com/spf13/pflag"

	_ "github.com/go-sql-driver/mysql" // import mysql driver
	_ "github.com/mattn/go-sqlite3"    // import sqlite3 driver
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite" // import CGo-free port of SQLite/SQLite3.
)

// GenerateCmd is the struct representing generate sub-command.
type GenerateCmd struct {
	NumRecs   int
	BatchSize int
	Vacuum    bool
	// Prepared use sql.DB Prepared statement for later queries or executions.
	Prepared   bool
	LogSeconds int

	currentSeq *atomic.Uint32
}

// nolint:gochecknoinits
func init() {
	c := GenerateCmd{}
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "generate records to benchmark against",
		Long: `This command generates records to benchmark against.
Each record consists of an ID, a 8 byte hex encoded random value
and a SHA256 hash of said random value.

ATTENTION: The 'bench' table will be DROPPED each time this command is called, before it
is (re)-generated!
	`,
		Run: c.run,
	}

	rootCmd.AddCommand(cmd)
	c.initFlags(cmd.Flags())
}

func (g *GenerateCmd) initFlags(f *pflag.FlagSet) {
	// Here you will define your flags and configuration settings.
	f.IntVarP(&g.NumRecs, "records", "r", 1000, "number of records to generate")
	f.IntVarP(&g.BatchSize, "batch", "b", 100, "number of records as a batch to insert at one time")
	f.IntVarP(&g.LogSeconds, "interval", "i", 2, "interval seconds between progress messages")
	f.BoolVarP(&g.Vacuum, "vacuum", "v", false, "VACUUM database file after the records generated.")
	f.BoolVarP(&g.Prepared, "prepared", "p", false, "use sql.DB Prepared statement for later queries or executions.")
}

func (g *GenerateCmd) run(cmd *cobra.Command, args []string) {
	log.Printf("Generating records by config %+v", g)

	db := setupBench(true, 1)
	defer db.Close()

	// Preinitialize i so that we can use it in a goroutine to give proper feedback
	g.currentSeq = atomic.NewUint32(0)
	// Set up logging mechanism. We use a goroutine here which logs the
	// records already generated every two seconds until "done" is signaled
	// via the channel.
	done := make(chan bool)
	start := time.Now()

	if g.NumRecs > 0 {
		go g.inserts(db, done)
		g.progressLogging(start, done)
	}

	if g.Vacuum {
		vacuumDB(db)
	}
}

// nolint:gomnd,gosec
func (g *GenerateCmd) inserts(db *sql.DB, done chan bool) {
	t, ok := tables[table]
	if !ok {
		log.Fatalf("%s does not exist", table)
	}

	// MySQL 协议定义两个字节表示传输参数个数，因此最大值是2^16-1=65536
	// 防止出现错误：failed Error 1390: Prepared statement contains too many placeholders
	// The prepared statement protocol defines a signed `2 bytes` short to describe the number of parameters
	// that will be retrieved from the server. The client firstly calls COM_STMT_PREPARE,
	// for which it receives a COM_STMT_PREPARE response if successful.
	if driverName == "mysql" && g.BatchSize*t.InsertFieldsNum > 65535 {
		batchSize := 65535 / t.InsertFieldsNum
		log.Printf("adjust batch size to %d", batchSize)
		g.BatchSize = batchSize
	}

	// Prepare values needed so that there aren't any allocations done in the loop
	query := t.CreateInsertSQL(g.BatchSize)
	execFn := func(args ...interface{}) (sql.Result, error) { return db.Exec(query, args...) }

	if g.Prepared {
		ps, err := db.Prepare(query)
		if err != nil {
			log.Fatalf("prepare len:%d query %s error %s", len(query), abbreviate(query, 1000), err)
		}

		defer ps.Close()

		execFn = ps.Exec
	}

	lastNum := g.NumRecs % g.BatchSize

	// Start generation of actual records
	log.Print("Starting inserts")

	args := make([]interface{}, 0, g.BatchSize*3)

	g.currentSeq.Load()

	for i := int(g.currentSeq.Load()); i < g.NumRecs; i = int(g.currentSeq.Add(1)) {
		args = append(args, t.Generator(i)...)

		if len(args) == g.BatchSize*t.InsertFieldsNum {
			if _, err := execFn(args...); err != nil {
				log.Fatalf("Inserting values into database failed: %s", err)
			}

			args = args[0:0]
		} else if lastNum > 0 && i+1 == g.NumRecs {
			query := t.CreateInsertSQL(lastNum)
			if _, err := db.Exec(query, args...); err != nil {
				log.Fatalf("Inserting values into database failed: %s", err)
			}
		}
	}

	// Signal the progress log that we are done
	done <- true
}

func abbreviate(s string, maxSize int) string {
	if len(s) < maxSize {
		return s
	}

	if maxSize <= 3 {
		return "..."[:maxSize]
	}

	return s[:maxSize-3] + "..."
}

// nolint:gomnd
func (g *GenerateCmd) progressLogging(start time.Time, done chan bool) {
	log.Print("Starting progress logging")

	l := len(fmt.Sprintf("%d", g.NumRecs))
	// Precalculate the percentage each record represents
	p := float64(100) / float64(g.NumRecs)

	ticker := time.NewTicker(time.Duration(g.LogSeconds) * time.Second)
	defer ticker.Stop()

out:
	for {
		select {
		// Since this is a time consuming process depending on the number of
		// records	created, we want some feedback every 2 seconds
		case <-ticker.C:
			dur := time.Since(start)
			i := g.currentSeq.Load()
			log.Printf("%*d/%*d (%6.2f%%) written in %s, avg: %s/record, %2.2f records/s",
				l, i, l, g.NumRecs, p*float64(i), dur,
				time.Duration(dur.Nanoseconds()/int64(i)), float64(i)/dur.Seconds())
		case <-done:
			break out
		}
	}

	dur := time.Since(start)
	log.Printf("%*d/%*d (%6.2f%%) written in %s, avg: %s/record, %2.2f records/s",
		l, g.NumRecs, l, g.NumRecs, p*float64(g.NumRecs), dur,
		time.Duration(dur.Nanoseconds()/int64(g.NumRecs)), float64(g.NumRecs)/dur.Seconds())
}

func vacuumDB(db *sql.DB) {
	log.Print("Vaccum database file")

	start := time.Now()

	if _, err := db.Exec("VACUUM"); err != nil {
		log.Printf("Vacuum database caused an error: %s", err)
		log.Print("Proceed with according caution.")
	}

	since := time.Since(start)
	log.Printf("Vacuum took %s", since)
}
