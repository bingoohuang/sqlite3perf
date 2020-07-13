# sqlite3perf

This repository is originally forked from [mwmahlberg/sqlite3perf](https://github.com/mwmahlberg/sqlite3perf).

## Inserts performance among different batch size(prepared mode)

batchSize | cost of 10000 rows inserts | records/s
---|---|---
10|465ms|21476.82 records/s
100|116ms|86119.24 records/s
500|65ms|152534.94 records/s
1000|59ms|168162.75 records/s

```bash
$ sqlite3perf generate -r 10000 -b 10
2020/07/09 23:08:39 Generating 10000 records
2020/07/09 23:08:39 Opening database
2020/07/09 23:08:39 Dropping table 'bench' if already present
2020/07/09 23:08:39 (Re-)creating table 'bench'
2020/07/09 23:08:39 Setting up the environment
2020/07/09 23:08:39 Starting progress logging
2020/07/09 23:08:39 Starting inserts
2020/07/09 23:08:40 10000/10000 (100.00%) written in 465.618282ms, avg: 46.561µs/record, 21476.82 records/s

# bingoo @ 192 in ~/GitHub/sqlite3perf on git:master x [23:08:40]
$ sqlite3perf generate -r 10000 -b 100
2020/07/09 23:08:43 Generating 10000 records
2020/07/09 23:08:43 Opening database
2020/07/09 23:08:43 Dropping table 'bench' if already present
2020/07/09 23:08:43 (Re-)creating table 'bench'
2020/07/09 23:08:43 Setting up the environment
2020/07/09 23:08:43 Starting progress logging
2020/07/09 23:08:43 Starting inserts
2020/07/09 23:08:43 10000/10000 (100.00%) written in 116.118071ms, avg: 11.611µs/record, 86119.24 records/s

# bingoo @ 192 in ~/GitHub/sqlite3perf on git:master x [23:08:43]
$ sqlite3perf generate -r 10000 -b 500
2020/07/09 23:08:48 Generating 10000 records
2020/07/09 23:08:48 Opening database
2020/07/09 23:08:48 Dropping table 'bench' if already present
2020/07/09 23:08:48 (Re-)creating table 'bench'
2020/07/09 23:08:48 Setting up the environment
2020/07/09 23:08:48 Starting progress logging
2020/07/09 23:08:48 Starting inserts
2020/07/09 23:08:48 10000/10000 (100.00%) written in 65.55875ms, avg: 6.555µs/record, 152534.94 records/s

# bingoo @ 192 in ~/GitHub/sqlite3perf on git:master x [23:08:48]
$ sqlite3perf generate -r 10000 -b 1000
2020/07/09 23:08:55 Generating 10000 records
2020/07/09 23:08:55 Opening database
2020/07/09 23:08:55 Dropping table 'bench' if already present
2020/07/09 23:08:55 (Re-)creating table 'bench'
2020/07/09 23:08:55 Setting up the environment
2020/07/09 23:08:55 Starting progress logging
2020/07/09 23:08:55 Starting inserts
2020/07/09 23:08:55 10000/10000 (100.00%) written in 59.466201ms, avg: 5.946µs/record, 168162.75 records/s
```

## Compare between prepared and non-prepared

mode | cost
---|---
prepared|128387.08 records/s
non|104741.21 records/s

```bash
$ sqlite3perf generate -r 30000 -p                           [五  7/10 09:57:17 2020]
2020/07/10 09:57:27 Generating records by config &{NumRecs:30000 BatchSize:100 Vacuum:false Prepared:true LogSeconds:2 cmd:0x4ae0680}
2020/07/10 09:57:27 Opening database
2020/07/10 09:57:27 Dropping table 'bench' if already present
2020/07/10 09:57:27 (Re-)creating table 'bench'
2020/07/10 09:57:27 Setting up the environment
2020/07/10 09:57:27 Starting progress logging
2020/07/10 09:57:27 Starting inserts
2020/07/10 09:57:27 30000/30000 (100.00%) written in 233.668369ms, avg: 7.788µs/record, 128387.08 records/s

sqlite3perf on  master [!] via 🐹 v1.14.4 via 🐍 v2.7.16
$ sqlite3perf generate -r 30000                              [五  7/10 09:57:27 2020]
2020/07/10 09:57:30 Generating records by config &{NumRecs:30000 BatchSize:100 Vacuum:false Prepared:false LogSeconds:2 cmd:0x4ae0680}
2020/07/10 09:57:30 Opening database
2020/07/10 09:57:30 Dropping table 'bench' if already present
2020/07/10 09:57:30 (Re-)creating table 'bench'
2020/07/10 09:57:30 Setting up the environment
2020/07/10 09:57:30 Starting progress logging
2020/07/10 09:57:30 Starting inserts
2020/07/10 09:57:30 30000/30000 (100.00%) written in 286.420228ms, avg: 9.547µs/record, 104741.21 records/s

sqlite3perf on  master [!] via 🐹 v1.14.4 via 🐍 v2.7.16
```

## different connect options.

Options |Prepared| speed(records/s)
---     |---     |---
`_sync=0&mode=memory&cache=shared`|yes |176285.79
`_sync=0&mode=memory&cache=shared`|no  |134848.84
`_sync=0&mode=memory`             |yes |173415.60
`_sync=0&mode=memory`             |no  |137106.92
`_sync=0`                         |yes |176476.27
`_sync=0`                         |no  |138742.34
(none)                          |yes |135183.90
(none)                          |no  |107080.79

> As the data showed, the `_sync=0` with `Prepared` reached the max speed.

[_sync=0 `PRAGMA synchronous = 0 | OFF`](https://www.sqlite.org/pragma.html#pragma_synchronous):

SQLite continues without syncing as soon as it has handed data off to the operating system.
If the application running SQLite crashes, the data will be safe,
but the database might become corrupted if the operating system crashes or the computer loses power
before that data has been written to the disk surface. On the other hand,
commits can be orders of magnitude faster with synchronous OFF.

[mode=memory SQLITE_OPEN_MEMORY](https://www.sqlite.org/c3ref/open.html)

The database will be opened as an in-memory database.
The database is named by the "filename" argument for the purposes of cache-sharing,
if shared cache mode is enabled, but the "filename" is otherwise ignored.

[cache=shared SQLite Shared-Cache Mode](https://www.sqlite.org/sharedcache.html)

Starting with version 3.3.0 (2006-01-11), SQLite includes a special "shared-cache" mode (disabled by default)
intended for use in embedded servers. If shared-cache mode is enabled and a thread
establishes multiple connections to the same database, the connections share a single data and schema cache.
This can significantly reduce the quantity of memory and IO required by the system.

```bash
$ sqlite3perf generate -r 50000 -b 100 -o "?_sync=0&mode=memory&cache=shared"
2020/07/11 00:20:14 Generating records by config &{NumRecs:50000 BatchSize:100 Vacuum:false Prepared:false LogSeconds:2 Options:?_sync=0&mode=memory&cache=shared cmd:0x4ae0680}
2020/07/11 00:20:14 Opening database
2020/07/11 00:20:14 Dropping table 'bench' if already present
2020/07/11 00:20:14 (Re-)creating table 'bench'
2020/07/11 00:20:14 Setting up the environment
2020/07/11 00:20:14 Starting progress logging
2020/07/11 00:20:14 Starting inserts
2020/07/11 00:20:15 50000/50000 (100.00%) written in 349.308517ms, avg: 6.986µs/record, 143139.94 records/s

# bingoo @ 192 in ~/GitHub/sqlite3perf on git:master x [0:20:15]
$ sqlite3perf generate -r 50000 -b 100 -o "?_sync=0&mode=memory"
2020/07/11 00:20:21 Generating records by config &{NumRecs:50000 BatchSize:100 Vacuum:false Prepared:false LogSeconds:2 Options:?_sync=0&mode=memory cmd:0x4ae0680}
2020/07/11 00:20:21 Opening database
2020/07/11 00:20:21 Dropping table 'bench' if already present
2020/07/11 00:20:21 (Re-)creating table 'bench'
2020/07/11 00:20:21 Setting up the environment
2020/07/11 00:20:21 Starting progress logging
2020/07/11 00:20:21 Starting inserts
2020/07/11 00:20:21 50000/50000 (100.00%) written in 366.342047ms, avg: 7.326µs/record, 136484.47 records/s

# bingoo @ 192 in ~/GitHub/sqlite3perf on git:master x [0:20:22]
$ sqlite3perf generate -r 50000 -b 100 -o "?_sync=0"
2020/07/11 00:20:29 Generating records by config &{NumRecs:50000 BatchSize:100 Vacuum:false Prepared:false LogSeconds:2 Options:?_sync=0 cmd:0x4ae0680}
2020/07/11 00:20:29 Opening database
2020/07/11 00:20:29 Dropping table 'bench' if already present
2020/07/11 00:20:29 (Re-)creating table 'bench'
2020/07/11 00:20:29 Setting up the environment
2020/07/11 00:20:29 Starting progress logging
2020/07/11 00:20:29 Starting inserts
2020/07/11 00:20:30 50000/50000 (100.00%) written in 376.316893ms, avg: 7.526µs/record, 132866.74 records/s

# bingoo @ 192 in ~/GitHub/sqlite3perf on git:master x [0:20:30]
$ sqlite3perf generate -r 50000 -b 100
2020/07/11 00:20:41 Generating records by config &{NumRecs:50000 BatchSize:100 Vacuum:false Prepared:false LogSeconds:2 Options: cmd:0x4ae0680}
2020/07/11 00:20:41 Opening database
2020/07/11 00:20:41 Dropping table 'bench' if already present
2020/07/11 00:20:41 (Re-)creating table 'bench'
2020/07/11 00:20:41 Setting up the environment
2020/07/11 00:20:41 Starting progress logging
2020/07/11 00:20:41 Starting inserts
2020/07/11 00:20:41 50000/50000 (100.00%) written in 447.981594ms, avg: 8.959µs/record, 111611.73 records/s
```

## [Command Line Shell For SQLite](https://www.sqlite.org/cli.html)

```bash
$ sqlite3 sqlite3perf.db
SQLite version 3.28.0 2019-04-15 14:49:49
Enter ".help" for usage hints.
sqlite> .tables
bench
sqlite> .schema bench
CREATE TABLE bench(ID int PRIMARY KEY ASC, rand TEXT, hash TEXT);
sqlite> select * from bench limit 3;
0|70d2e0802359c436|b3085192086ceeeeaa2ec20f3ccc9047f3148cd3154ae734ec93adc4ab5661f2
1|4125c6f752726494|7003a29fa88c302e35b04b9a3011e8b67bcf970f3b9c18bdd26227eff0ea6268
2|85be8e175929949f|f9c2fda0688eb6fda643178f80e4500faddf39ff9f6e7c209319106b16057f68
sqlite> select count(*) from bench;
50000
sqlite> .header on
sqlite> .mode column
sqlite> select count(*) from bench;
count(*)
----------
50000
sqlite> select * from bench limit 3;
ID          rand              hash
----------  ----------------  ----------------------------------------------------------------
0           70d2e0802359c436  b3085192086ceeeeaa2ec20f3ccc9047f3148cd3154ae734ec93adc4ab5661f2
1           4125c6f752726494  7003a29fa88c302e35b04b9a3011e8b67bcf970f3b9c18bdd26227eff0ea6268
2           85be8e175929949f  f9c2fda0688eb6fda643178f80e4500faddf39ff9f6e7c209319106b16057f68
sqlite> .quit
```

## [How to Create an Efficient Pagination in SQL (PoC)](https://github.com/IvoPereira/Efficient-Pagination-SQL-PoC)

- [Why You Shouldn't Use OFFSET and LIMIT For Your Pagination](https://ivopereira.net/content/efficient-pagination-dont-use-offset-limit)
- [db fiddle](https://www.db-fiddle.com/f/3JSpBxVgcqL3W2AzfRNCyq/1)

![image](https://user-images.githubusercontent.com/1940588/87237941-eeabd600-c42e-11ea-9565-865a3e37921e.png)

The following is result that I run the same POC on sqlite3:

```bash
🕙[2020-07-12 10:48:27.217] ❯ sqlite3perf generate -r 10000000 -b 2000 -o "?_sync=0" -p
2020/07/12 10:48:28 Generating records by config &{NumRecs:10000000 BatchSize:2000 Vacuum:false Prepared:true LogSeconds:2 Options:?_sync=0 cmd:0x4ae0680}
2020/07/12 10:48:28 Opening database
2020/07/12 10:48:28 Dropping table 'bench' if already present
2020/07/12 10:48:28 (Re-)creating table 'bench'
2020/07/12 10:48:28 Setting up the environment
2020/07/12 10:48:28 Starting progress logging
2020/07/12 10:48:28 Starting inserts
2020/07/12 10:48:30   845999/10000000 (  8.46%) written in 2.000076737s, avg: 2.364µs/record, 422983.27 records/s
2020/07/12 10:48:32  1701999/10000000 ( 17.02%) written in 4.000113037s, avg: 2.35µs/record, 425487.73 records/s
2020/07/12 10:48:34  2563999/10000000 ( 25.64%) written in 6.000478095s, avg: 2.34µs/record, 427299.12 records/s
2020/07/12 10:48:36  3417999/10000000 ( 34.18%) written in 8.000240041s, avg: 2.34µs/record, 427237.06 records/s
2020/07/12 10:48:38  4263999/10000000 ( 42.64%) written in 10.000036515s, avg: 2.345µs/record, 426398.34 records/s
2020/07/12 10:48:40  5115607/10000000 ( 51.16%) written in 12.000338739s, avg: 2.345µs/record, 426288.63 records/s
2020/07/12 10:48:42  5969999/10000000 ( 59.70%) written in 14.000139782s, avg: 2.345µs/record, 426424.24 records/s
2020/07/12 10:48:44  6821999/10000000 ( 68.22%) written in 16.000071248s, avg: 2.345µs/record, 426373.04 records/s
2020/07/12 10:48:46  7673999/10000000 ( 76.74%) written in 18.000129294s, avg: 2.345µs/record, 426330.22 records/s
2020/07/12 10:48:48  8529999/10000000 ( 85.30%) written in 20.000615404s, avg: 2.344µs/record, 426486.83 records/s
2020/07/12 10:48:50  9337999/10000000 ( 93.38%) written in 22.000119926s, avg: 2.355µs/record, 424452.19 records/s
2020/07/12 10:48:52 10000000/10000000 (100.00%) written in 23.55338367s, avg: 2.355µs/record, 424567.45 records/s

~/Downloads via 🐹 v1.14.4 took 23s
🕙[2020-07-12 10:48:52.438] ❯ time sqlite3 sqlite3perf.db  "select * from bench limit 5 offset 2850001"
2850001|454973fc69f76679|a4a3188c7554ebefb8d5749399ad23c593a4dd0dadef8a235b308cc808c194a7
2850002|ab4a4f8d63461ae8|83b1ffa19dff7a2fd07459b5b2f9aac8e4e14f3fa1584f8d1ed6bca2c37eaef1
2850003|01629449faf2ae99|e124f6cac5d6a22af62851c1dc547853a97e923a5a7dc8d476f81d355cc522b8
2850004|15aa9d95a4acf49a|dda4d551ba8c3bf9ac10e22682c614ad331eb4fecea4daa8c6dbe8cd026dfda4
2850005|47a4d0a49b83a921|373d0bc79d4060c4a0c7b8183592b8ce3732f34e159d8d7bb5ffb7f9ab2a4288
sqlite3 sqlite3perf.db "select * from bench limit 5 offset 2850001"  0.08s user 0.06s system 98% cpu 0.145 total

~/Downloads via 🐹 v1.14.4
🕙[2020-07-12 10:49:06.711] ❯ time sqlite3 sqlite3perf.db  "select * from bench where id > 2850000 limit 5"
2850001|454973fc69f76679|a4a3188c7554ebefb8d5749399ad23c593a4dd0dadef8a235b308cc808c194a7
2850002|ab4a4f8d63461ae8|83b1ffa19dff7a2fd07459b5b2f9aac8e4e14f3fa1584f8d1ed6bca2c37eaef1
2850003|01629449faf2ae99|e124f6cac5d6a22af62851c1dc547853a97e923a5a7dc8d476f81d355cc522b8
2850004|15aa9d95a4acf49a|dda4d551ba8c3bf9ac10e22682c614ad331eb4fecea4daa8c6dbe8cd026dfda4
2850005|47a4d0a49b83a921|373d0bc79d4060c4a0c7b8183592b8ce3732f34e159d8d7bb5ffb7f9ab2a4288
sqlite3 sqlite3perf.db "select * from bench where id > 2850000 limit 5"  0.00s user 0.00s system 70% cpu 0.006 total

~/Downloads via 🐹 v1.14.4
🕙[2020-07-12 10:49:12.989] ❯
```

## Original blog content

This repository contains a small application which was created while researching a proper
answer to the question [Faster sqlite 3 query in go? I need to process 1million+ rows as fast as possible][so:oq].

The assumption there was that Python is faster with accessing SQLite3 than Go is.

I wanted to check this and hence I wrote a generator for entries into an SQLite database as well as a Go implementation and a Python implementation of a simple access task:

1. Read all rows from table bench, which consists of an ID, a hex encoded 8 byte random value and a hex encoded SHA256 hash of said random values.
2. Create a SHA256 hex encoded checksum from the decoded random value of a row.
3. Compare the stored hash value against the generated one.
4. If they match, continue, otherwise throw an error.

[so:oq]: https://stackoverflow.com/questions/48000940/

## Introduction

My assumption was that we have a problem with how the performance is measured here, so I wrote a little Go program to generate records and save them into a SQLite database as well as a Python and Go implementation of a little task to do on those records.

You can find the according repository at [mwmahlberg/sqlite3perf](https://github.com/mwmahlberg/sqlite3perf)

### The data model

The records generated consist of

- ID: [A row ID generated by SQLite](https://sqlite.org/lang_createtable.html#rowid)
- rand: A hex encoded, 8 byte, pseudo-random value
- hash: A hex encoded, SHA256 hash of the unencoded rand

The table's schema is relatively simple:

```sql
$ sqlite3 sqlite3perf.db
SQLite version 3.28.0 2019-04-15 14:49:49
Enter ".help" for usage hints.
sqlite> .schema
CREATE TABLE bench (ID int PRIMARY KEY ASC, rand TEXT, hash TEXT);
```

First I generated 1.5M records and vacuumed the sqlite database afterwards with

```bash
$ sqlite3perf generate -r 1500000 -v
2020/07/09 22:34:44 Generating 1500000 records
2020/07/09 22:34:44 Opening database
2020/07/09 22:34:44 Dropping table 'bench' if already present
2020/07/09 22:34:44 (Re-)creating table 'bench'
2020/07/09 22:34:44 Setting up the environment
2020/07/09 22:34:44 Starting progress logging
2020/07/09 22:34:44 Starting inserts
2020/07/09 22:34:46  240099/1500000 ( 16.01%) written in 2.000146054s, avg: 8.33µs/record, 120040.73 records/s
2020/07/09 22:34:48  488499/1500000 ( 32.57%) written in 4.000127495s, avg: 8.188µs/record, 122120.86 records/s
2020/07/09 22:34:50  730799/1500000 ( 48.72%) written in 6.000064551s, avg: 8.21µs/record, 121798.52 records/s
2020/07/09 22:34:52  968999/1500000 ( 64.60%) written in 8.000359735s, avg: 8.256µs/record, 121119.43 records/s
2020/07/09 22:34:54 1200899/1500000 ( 80.06%) written in 10.000176936s, avg: 8.327µs/record, 120087.78 records/s
2020/07/09 22:34:56 1430799/1500000 ( 95.39%) written in 12.000074408s, avg: 8.386µs/record, 119232.51 records/s
2020/07/09 22:34:56 1500000/1500000 (100.00%) written in 12.537909385s, avg: 8.358µs/record, 119637.17 records/s
2020/07/09 22:34:56 Vaccumating database file
2020/07/09 22:34:58 Vacuumation took 2.070888698s
```

Next I called the Go implementation against those 1.5M records. Both the Go as well as the Python implementation
basically do the same simple task:

1. Read all entries from the database.
1. For each row, decode the random value from hex, then create a SHA256 hex from the result.
1. Compare the generated SHA256 hex string against the one stored in the database
1. If they match, continue, otherwise break.

### Assumptions

My assumption explicitly was that Python did some type of lazy loading and/or possibly even execution of the SQL query.

## The results

### Go implementation

```bash
$ sqlite3perf bench
2020/07/09 22:37:23 Running benchmark
2020/07/09 22:37:23 Time after query: 1.261861ms
2020/07/09 22:37:23 Beginning loop
2020/07/09 22:37:23 Acessing the first result set
        ID 0,
        rand: 819f4b54a911924d,
        hash: 507d24d4ae8ec1b7c89939abc6c80959ce7f04334c6d9c3b15ac86c7aaef24da
took 123.618µs
2020/07/09 22:37:25 1,101,829 rows processed
2020/07/09 22:37:26 1,500,000 rows processed
2020/07/09 22:37:26 Finished loop after 2.71359178s
2020/07/09 22:37:26 Average 1.809µs per record, 2.714910396s overall
```

Note the values for "time after query" ( the time the query command took to return)
and the time it took to access the first result set after the iteration over the result set was started.

### Python implementation

```bash
$ python bench.py
07/09/2020 22:38:19 Starting up
07/09/2020 22:38:19 Time after query: 232µs
07/09/2020 22:38:19 Beginning loop
07/09/2020 22:38:20 Accessing first result set
        ID: 0
        rand: 819f4b54a911924d
        hash: 507d24d4ae8ec1b7c89939abc6c80959ce7f04334c6d9c3b15ac86c7aaef24da
took 1.544171 s
07/09/2020 22:38:24 Finished loop after 5.655742s
07/09/2020 22:38:24 Average: 3.770µs per record, 0:00:05.656060 overall
```

Again, note the value for "time after query" and the time it took to access the first result set.

## Summary

It took the Go implementation quite a while to return after the SELECT query was send, while Python seemed to be blazing fast in comparison. However, from the time it took to actually access the first result set, we can see that the Go implementation is more than 500 times faster to actually access the first result set (5.372329ms vs 2719.312ms) and about double as fast for the task at hand as the Python implementation.

## Notes

- In order to prove the assumption that Python actually does lazy loading on the result set, each and every row and column had to be accessed in order to make sure that Python is forced to actually read the value from the database.
- I chose a hashing task because presumably the implementation of SHA256 is highly optimised in both languages.

## Conclusion

Python does seem to do lazy loading of result sets and possibly does not even execute a query unless the according result set
is actually accessed. In this simulated scenario, mattn's SQLite driver for Go outperforms Python's by between roughly 100%
and orders of magnitude, depending on what you want to do.

Edit: So in order to have a fast processing, implement your task in Go. While it takes longer to send the actual query,
accessing the individual rows of the result set is by far faster. I'd suggest starting out with a small subset of your data,
say 50k records. Then, to further improve your code, use [profiling](https://blog.golang.org/profiling-go-programs)
to identify your bottlenecks.

Depending on what you want to do during processing, [pipelines](https://blog.golang.org/pipelines) for example might help,
but how to improve the processing
speed of the task at hand is difficult to say without actual code or a thorough description.
