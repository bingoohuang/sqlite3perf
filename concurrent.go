package sqlite3perf

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/spf13/pflag"

	"github.com/spf13/cobra"
)

// ConcurrentCmd is the struct representing to run concurrent reads and writes sub-command.
type ConcurrentCmd struct {
	clear    bool
	close    bool
	reads    int
	from     int64
	writes   int
	maxConns int
	duration time.Duration

	w, r int64
}

// nolint:gochecknoinits
func init() {
	c := ConcurrentCmd{}
	cmd := &cobra.Command{
		Use:   "concurrent",
		Short: "concurrent reads and writes test",
		Long:  `verify the supporting condition for concurrent reads and writes of sqlite3.`,
		Run:   c.run,
	}

	rootCmd.AddCommand(cmd)
	c.initFlags(cmd.Flags())
}

func (g *ConcurrentCmd) initFlags(f *pflag.FlagSet) {
	// Here you will define your flags and configuration settings.
	f.BoolVar(&g.clear, "clear", false, "clear the database at the startup")
	f.BoolVar(&g.close, "close", true, "close the database at the end")
	f.IntVarP(&g.reads, "reads", "r", 100, "number of goroutines to read")
	f.IntVarP(&g.writes, "writes", "w", 100, "number of goroutines to write")
	f.Int64Var(&g.from, "from", 0, "ID from for writes")
	f.IntVarP(&g.maxConns, "maxConns", "m", 1, "max of open connections to db.")
	f.DurationVarP(&g.duration, "duration", "d", 60*time.Second, "duration to run")
}

func (g *ConcurrentCmd) run(cmd *cobra.Command, args []string) {
	log.Printf("concurrent reads and writes verifying")

	db := setupBench(g.clear, g.maxConns)
	if g.close {
		defer db.Close()
	}

	closeCh := make(chan bool)
	quitCh := make(chan bool)

	ctx, _ := context.WithTimeout(cmd.Context(), g.duration)

	for i := 0; i < g.reads; i++ {
		go g.read(ctx, db, closeCh, quitCh)
	}

	atomic.StoreInt64(&g.w, g.from)

	for i := 0; i < g.writes; i++ {
		go g.write(ctx, db, closeCh, quitCh)
	}

	<-ctx.Done()

	log.Printf("notify all reads and writes goroutines to exit")
	close(closeCh)

	for i := 0; i < g.reads+g.writes; i++ {
		<-quitCh
	}

	log.Printf("all reads and writes goroutines exited")
}

func (g *ConcurrentCmd) write(ctx context.Context, db *sql.DB, closeCh, quitCh chan bool) {
	h := NewHasher()
	defer func() {
		quitCh <- true
	}()

	for goon(ctx, closeCh) {
		s, sum := h.Gen()
		wc := atomic.AddInt64(&g.w, 1)
		// log.Printf("insert ID:%d, rand:%s, hash:%s", id, s, sum)
		if _, err := db.ExecContext(ctx, "insert into bench(id, rand, hash) values(?, ?, ?)", wc, s, sum); err != nil {
			if ctx.Err() != nil {
				return
			}

			if errs := err.Error(); strings.Contains(errs, "UNIQUE constraint failed:") {
				log.Printf("Inserting values into database failed: %s", err)
			} else {
				log.Fatalf("Inserting values into database failed: %s", err)
			}
		}

		rc := wc - g.from
		if rc%10000 == 0 {
			log.Printf("%d rows written", rc)
			SleepContext(ctx, 1*time.Second)
		}
	}
}

func (g *ConcurrentCmd) read(ctx context.Context, db *sql.DB, closeCh, quitCh chan bool) {
	var (
		ID   int64
		rand string
		hash string
	)

	defer func() {
		quitCh <- true
	}()

	for goon(ctx, closeCh) {
		rc := atomic.AddInt64(&g.r, 1)
		rows, err := db.QueryContext(ctx, "select * from bench order by ID desc limit 3")
		if ctx.Err() != nil {
			return
		}

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
			SleepContext(ctx, g.duration)
		}

		if err := rows.Close(); err != nil {
			log.Fatal(err)
		}
	}
}

func goon(ctx context.Context, closeCh chan bool) bool {
	select {
	case <-closeCh:
		return false
	case <-ctx.Done():
		return false
	default:
		return true
	}
}
