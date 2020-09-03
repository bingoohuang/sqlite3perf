package sqlite3perf

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
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

	// trap Ctrl+C and call cancel on the context
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	db := setupBench(g.clear, g.maxConns)
	if g.close {
		defer db.Close()
	}

	closeCh := make(chan bool)
	quitCh := make(chan bool)

	for i := 0; i < g.reads; i++ {
		go g.read(db, closeCh, quitCh, c)
	}

	atomic.StoreInt64(&g.w, g.from)

	for i := 0; i < g.writes; i++ {
		go g.write(db, closeCh, quitCh, c)
	}

	select {
	case <-c:
		log.Printf("interrupt signal catched")
	case <-time.After(g.duration):
		log.Printf("sleeped %s", g.duration)
	}

	log.Printf("notify all reads and writes goroutines to exit")
	close(closeCh)

	for i := 0; i < g.reads+g.writes; i++ {
		<-quitCh
	}

	log.Printf("all reads and writes goroutines exited")
}

func (g *ConcurrentCmd) write(db *sql.DB, closeCh, quitCh chan bool, c chan os.Signal) {
	h := NewHasher()
	defer func() {
		quitCh <- true
	}()

	for goon(closeCh, c) {
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

func (g *ConcurrentCmd) read(db *sql.DB, closeCh, quitCh chan bool, c chan os.Signal) {
	var (
		ID   int64
		rand string
		hash string
	)

	defer func() {
		quitCh <- true
	}()

	for goon(closeCh, c) {
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

func goon(closeCh chan bool, c chan os.Signal) bool {
	select {
	case <-closeCh:
		return false
	case <-c:
		return false
	default:
		return true
	}
}
