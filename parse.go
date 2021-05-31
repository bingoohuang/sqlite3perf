package sqlite3perf

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

// ParseCmd is the struct representing parse sub-command.
type ParseCmd struct {
	File  string
	Regex string
}

// nolint:gochecknoinits
func init() {
	c := ParseCmd{}
	cmd := &cobra.Command{
		Use:   "parse",
		Short: "parse input by pattern",
		Long:  `parse the input file by pattern and save group values into table`,
		Run:   c.run,
	}
	c.initFlags(cmd.Flags())
	rootCmd.AddCommand(cmd)
}

func (g *ParseCmd) initFlags(f *pflag.FlagSet) {
	// Here you will define your flags and configuration settings.
	f.StringVarP(&g.File, "file", "f", "", "file to parse")
	f.StringVarP(&g.Regex, "regex", "r", "", "file to parse")
}

func (g *ParseCmd) run(cmd *cobra.Command, args []string) {
	log.Printf("Parse records by config %+v", g)

	log.Print("Opening database")
	// Database Setup
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error while opening database '%s': %s", dbPath, err.Error())
	}
	defer db.Close()

	// 2021/05/29 13:09:46 Replay POST http://192.166.223.29:9090/solr/licenseIndex/update?wt=javabin&version=2, cost 1.256731096s, status: 200
	re := regexp.MustCompile(`(?P<time>.{19}) Replay POST .*? cost (?P<cost>.*?), status: (?P<status>\d+)`)

	createTable := `create table ` + table + `(`
	insertTable := `insert into ` + table + `(`
	columns := 0
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			if name == "cost" {
				createTable += `costi REAL,`
				insertTable += `costi,`
				columns++
			}

			createTable += name + ` TEXT,`
			insertTable += name + `,`
			columns++
		}
	}
	createTable = createTable[:len(createTable)-1] + `)`
	insertTable = insertTable[:len(insertTable)-1] + `) values (` + strings.Repeat(`,?`, columns)[1:] + `)`
	if _, err := db.Exec(createTable); err != nil {
		log.Printf("create table %s: %s", createTable, err)
	}

	ps, err := db.Prepare(insertTable)
	if err != nil {
		log.Fatalf("prepare  %s error %s", insertTable, err)
	}

	f, err := os.Open(g.File)
	if err != nil {
		log.Fatalf("open %s error: %v", g.File, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(ScanLines)
	totalRows, rows := 0, 0

	start := time.Now()

	ctx := cmd.Context()
	for scanner.Scan() && ctx.Err() == nil {
		data := scanner.Bytes()
		match := re.FindStringSubmatch(string(data))

		if len(match) == 0 {
			continue
		}

		result := make([]interface{}, 0, columns)
		for i, name := range re.SubexpNames() {
			if i != 0 && name != "" {
				if name == "cost" {
					duration, _ := time.ParseDuration(match[i])
					result = append(result, duration.Seconds())
				}

				result = append(result, match[i])
			}
		}
		if _, err := ps.ExecContext(ctx, result...); err != nil {
			log.Fatalf("exec error: %v", err)
		}
		rows++
		totalRows++
		if rows >= 1000 {
			Printf("Rows %d insertted, cost %s", totalRows, time.Since(start))
			rows = 0
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		log.Fatalf("scanner error: %v", err)
	}

	Printf("Rows %d insertted, cost %s", totalRows, time.Since(start))
}

var printN int

func Printf(format string, a ...interface{}) {
	if printN > 0 {
		fmt.Print(strings.Repeat("\b", printN))
	}
	printN, _ = fmt.Printf(format, a...)
}

var timePattern = regexp.MustCompile(`\d{4}[/-]\d\d[/-]\d\d \d\d:\d\d:\d\d `) // 2021/05/29 13:09:46

func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	const offset = 20

	if loc := timePattern.FindSubmatchIndex(data); len(loc) > 0 {
		if loc[0] > 0 {
			return loc[0], data[:loc[0]], nil
		} else if loc[0] == 0 && len(data) >= offset {
			if loc2 := timePattern.FindSubmatchIndex(data[offset:]); len(loc2) > 0 {
				i := offset + loc2[0]
				return i, data[:i], nil
			}
		}
	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}
