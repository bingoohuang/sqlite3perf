package sqlite3perf

import (
	"database/sql"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
)

// ConcurrentCmd is the struct representing to run concurrent reads and writes sub-command.
type ConcurrentCmd struct {
	clear    bool
	close    bool
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
	f.BoolVar(&g.clear, "clear", false, "clear the database at the startup")
	f.BoolVar(&g.close, "close", true, "close the database at the end")
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
	if g.close {
		defer db.Close()
	}

	closeCh := make(chan bool)
	quitCh := make(chan bool)

	for i := 0; i < g.reads; i++ {
		go g.read(db, closeCh, quitCh)
	}

	atomic.StoreInt64(&g.w, g.from)

	for i := 0; i < g.writes; i++ {
		go g.write(db, closeCh, quitCh)
	}

	time.Sleep(g.duration)
	log.Printf("notify all reads and writes goroutines to exit")

	close(closeCh)

	for i := 0; i < g.reads+g.writes; i++ {
		<-quitCh
	}

	log.Printf("all reads and writes goroutines exited")
}

func (g *ConcurrentCmd) write(db *sql.DB, closeCh, quitCh chan bool) {
	h := NewHasher()

	for {
		select {
		case <-closeCh:
			quitCh <- true
			return
		default:
		}

		s, sum := h.Gen()
		wc := atomic.AddInt64(&g.w, 1)
		// log.Printf("insert ID:%d, rand:%s, hash:%s", id, s, sum)
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

func (g *ConcurrentCmd) read(db *sql.DB, closeCh, quitCh chan bool) {
	var (
		ID   int64
		rand string
		hash string
	)

	for {
		select {
		case <-closeCh:
			quitCh <- true
			return
		default:
		}

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
