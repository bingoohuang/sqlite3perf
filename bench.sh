#!/bin/bash

set -x

rm -fr a.db* && sqlite3perf generate -r 50000 -b 100 --db "a.db?_journal=wal&mode=memory&sync=0" --prepared
rm -fr a.db* && sqlite3perf generate -r 50000 -b 100 --db "a.db?_journal=wal&sync=0" --prepared
rm -fr a.db* && sqlite3perf generate -r 50000 -b 100 --db "a.db?_journal=wal" --prepared
rm -fr a.db* && sqlite3perf generate -r 50000 -b 100 --db "a.db?_sync=0" --prepared
