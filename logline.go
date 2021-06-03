package sqlite3perf

import (
	"bufio"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"github.com/bingoohuang/gg/pkg/logline"
	"github.com/bingoohuang/gg/pkg/ss"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// ParseCmd is the struct representing parse sub-command.
type ParseCmd struct {
	File         string
	PatternFile  string
	QuoteReplace string
	LineStart    string
}

// nolint:gochecknoinits
func init() {
	c := ParseCmd{}
	cmd := &cobra.Command{
		Use:   "logline",
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
	f.StringVarP(&g.PatternFile, "pattern", "p", "", "pattern file ")
	f.StringVarP(&g.QuoteReplace, "quote", "", "\"", "quote replacement")
	f.StringVarP(&g.LineStart, "start", "", "2021/05/29 13:09:46", "line start")
}

func (g *ParseCmd) run(cmd *cobra.Command, args []string) {
	log.Printf("Parse records by config %+v", g)

	log.Print("Opening database")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error while opening database '%s': %v", dbPath, err)
	}
	defer db.Close()

	patternData, err := os.ReadFile(g.PatternFile)
	if err != nil {
		log.Fatalf("Error read file '%s': %v", g.PatternFile, err)
	}

	data := string(patternData)
	n := strings.IndexByte(data, '\n')
	if n < 0 {
		log.Fatalf("failed to read first line as samplee at '%s'", g.PatternFile)
	}
	samplee := data[:n]
	pattern := data[n+1:]
	if n := strings.IndexByte(pattern, '\n'); n > 0 {
		pattern = pattern[:n]
	}

	p, err := logline.NewPattern(samplee, pattern, logline.WithReplace(g.QuoteReplace, `"`))
	if err != nil {
		log.Fatalf("failed to parse logline pattern: %v", err)
	}

	createTable, insertTable, columns := g.createSqls(p)
	if _, err := db.Exec(createTable); err != nil {
		// maybe already created, just print error and continue.
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
	scanner.Split(NewScanLines(g.LineStart))
	totalRows, rows := 0, 0

	start := time.Now()
	ctx := cmd.Context()
	for scanner.Scan() && ctx.Err() == nil {
		line := bytes.TrimSpace(scanner.Bytes())
		m, ok := p.ParseBytes(line)
		if !ok {
			continue
		}

		result := make([]interface{}, 0, columns)
		for _, dot := range p.Dots {
			if dot.Valid() {
				result = append(result, m[dot.Name])
			}
		}
		if _, err := ps.ExecContext(ctx, result...); err != nil {
			log.Fatalf("exec error: %v", err)
		}
		rows++
		totalRows++
		if rows >= 1000 {
			Printing("Rows %d generated, cost %s", totalRows, time.Since(start))
			rows = 0
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		log.Fatalf("scanner error: %v", err)
	}

	PrintEnd("Rows %d generated, cost %s\n", totalRows, time.Since(start))
}

func (g *ParseCmd) createSqls(p *logline.Pattern) (createTable, insertTable string, columns int) {
	createTable = `create table ` + table + `(`
	insertTable = `insert into ` + table + `(`
	for _, dot := range p.Dots {
		if !dot.Valid() {
			continue
		}
		switch dot.Type {
		case logline.Digits:
			createTable += dot.Name + ` INTEGER,`
		case logline.Float:
			createTable += dot.Name + ` REAL,`
		default:
			createTable += dot.Name + ` TEXT,`
		}
		insertTable += dot.Name + `,`
		columns++
	}
	createTable = createTable[:len(createTable)-1] + `)`
	insertTable = insertTable[:len(insertTable)-1] + `) values (` + strings.Repeat(`,?`, columns)[1:] + `)`
	return createTable, insertTable, columns
}

func NewScanLines(start string) bufio.SplitFunc {
	re := convertDigits(start)
	startPattern := regexp.MustCompile(re)
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		return ScanLines(startPattern, data, atEOF)
	}
}

func convertDigits(start string) (re string) {
	for _, c := range start {
		re += ss.If(unicode.IsDigit(c), `\d`, string(c))
	}
	return
}

var printN int

func Printing(format string, a ...interface{}) {
	if printN > 0 {
		fmt.Print(strings.Repeat("\b", printN))
	}
	printN, _ = fmt.Printf(format, a...)
}

func PrintEnd(format string, a ...interface{}) {
	if printN > 0 {
		fmt.Print(strings.Repeat("\b", printN))
	}
	log.Printf(format, a...)
}

func ScanLines(startPattern *regexp.Regexp, data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	const offset = 20

	if loc := startPattern.FindSubmatchIndex(data); len(loc) > 0 {
		if loc[0] > 0 {
			return loc[0], data[:loc[0]], nil
		} else if loc[0] == 0 && len(data) >= offset {
			if loc2 := startPattern.FindSubmatchIndex(data[offset:]); len(loc2) > 0 {
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
