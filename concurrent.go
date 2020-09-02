package sqlite3perf

import (
	"database/sql"
	"github.com/spf13/cobra"
	"log"
	"strings"
	"sync/atomic"
	"time"
)

// ConcurrentCmd is the struct representing to run concurrent reads and writes sub-command.
type ConcurrentCmd struct {
	clear    bool
	reads    int
	from     int64
	writes   int
	duration time.Duration

	w, r int64
}

// nolint:gochecknoinits
func init() {
	cmd := &cobra.Command{
		Use:   "concurrent",
		Short: "concurrent reads and writes test",
		Long:  `verify the supporting condition for concurrent reads and writes of sqlite3.`,
	}
	c := ConcurrentCmd{}

	rootCmd.AddCommand(cmd)
	c.initFlags(cmd)
	cmd.Run = c.run
}

func (g *ConcurrentCmd) initFlags(cmd *cobra.Command) {
	// Here you will define your flags and configuration settings.
	f := cmd.Flags()
	f.BoolVar(&g.clear, "clear", false,
		"clear the database at startup")
	f.IntVarP(&g.reads, "reads", "r", 100,
		"number of goroutines to read")
	f.IntVarP(&g.writes, "writes", "w", 100,
		"number of goroutines to write")
	f.Int64Var(&g.from, "from", 0,
		"ID from for writes")
	f.DurationVarP(&g.duration, "duration", "d", 60*time.Second,
		"druation to run")
}

func (g *ConcurrentCmd) run(cmd *cobra.Command, args []string) {
	log.Printf("concurrent reads and writes verifying")

	db := setupBench(g.clear)
	defer db.Close()

	for i := 0; i < g.reads; i++ {
		go g.read(db)
	}

	atomic.StoreInt64(&g.w, g.from)

	for i := 0; i < g.writes; i++ {
		go g.write(db)
	}

	time.Sleep(g.duration)
}

func (g *ConcurrentCmd) write(db *sql.DB) {
	time.Sleep(1 * time.Millisecond)

	h := NewHash()

	for i := 1; ; i++ {
		s, sum := h.Gen()
		wc := atomic.AddInt64(&g.w, 1)
		//log.Printf("insert ID:%d, rand:%s, hash:%s", id, s, sum)
		if _, err := db.Exec("insert into bench(id, rand, hash) values(?, ?, ?)", wc, s, sum); err != nil {
			if errs := err.Error(); strings.Contains(errs, "UNIQUE constraint failed:") {
				log.Printf("Inserting values into database failed: %s", err)
			} else {
				log.Fatalf("Inserting values into database failed: %s", err)
			}
		}

		rc := wc - g.from
		if rc%10000 == 0 {
			log.Printf("%d rows written", rc)
			time.Sleep(1 * time.Second)
		}
	}
}

func (g *ConcurrentCmd) read(db *sql.DB) {
	var (
		ID   int64
		rand string
		hash string
	)

	for i := 1; ; i++ {
		rc := atomic.AddInt64(&g.r, 1)
		rows, err := db.Query("select * from bench order by ID desc limit 3")
		if err != nil {
			log.Fatal(err)
		}

		for rows.Next() {
			if err = rows.Scan(&ID, &rand, &hash); err != nil {
				log.Fatal(err)
			}
		}

		if rc%100000 == 0 {
			log.Printf("reads:%d, ID:%d, rand:%s, hash:%s", rc, ID, rand, hash)
			time.Sleep(1 * time.Second)
		}

		if err := rows.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
