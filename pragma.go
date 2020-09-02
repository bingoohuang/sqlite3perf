package sqlite3perf

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

// PragmaCmd is the struct representing pragma sub-command.
type PragmaCmd struct {
	// cmd represents the generate command
	cmd *cobra.Command
}

// nolint:gochecknoinits
func init() {
	cmd := PragmaCmd{
		// generateCmd represents the generate command
		cmd: &cobra.Command{
			Use:   "pragma",
			Short: "pragma to get or set",
			Long: `PRAGMA Statements specific to SQLite,
like:
1). sqlite3perf pragma synchronous auto_vacuum journal_mode
2). sqlite3perf pragma synchronous=0 auto_vacuum=NONE
`,
		},
	}

	rootCmd.AddCommand(cmd.cmd)
	cmd.cmd.Run = cmd.run
}

func (g *PragmaCmd) run(cmd *cobra.Command, args []string) {
	log.Printf("Generating records by config %+v", g)

	log.Println("Opening database")
	// Database Setup
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error while opening database '%s': %s", dbPath, err.Error())
	}
	defer db.Close()

	for _, v := range args {
		keyValues := strings.SplitN(v, "=", 2)
		key := keyValues[0]
		value := ""

		if len(keyValues) > 1 {
			value = keyValues[1]
		}

		if !queryPragma(db, key) || value == "" {
			continue
		}

		alterPragma(db, key, value, v)
		queryPragma(db, key)
	}
}

func alterPragma(db *sql.DB, key string, value string, v string) {
	_, err := db.Exec(fmt.Sprintf("PRAGMA %s=%s", key, value))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("set PRAGMA %s successfully", v)
}

func queryPragma(db *sql.DB, key string) bool {
	row := db.QueryRow("PRAGMA " + key)
	previousValue := ""
	err := row.Scan(&previousValue)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("Unknown PRAGMA %s", key)
			return false
		}

		log.Fatal(err)
	}

	log.Printf("get PRAGMA %s=%s", key, previousValue)

	return true
}
