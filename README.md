# csv-tidy

A command-line tool that normalizes messy CSV files. It reads CSV from a
file or stdin, checks it, and writes a cleaned-up version to stdout.

Real-world CSV exports are rarely well-formed: rows are missing a trailing
column, spreadsheet software leaves stray whitespace around values, someone
pasted in a file saved with the wrong encoding and a few bytes came out as
garbage. Most tools either silently paper over this (and you don't find out
until a downstream job breaks on row 40,000) or crash with an unhelpful
error. csv-tidy does neither by default: it tells you exactly which line is
broken and why, and only fixes things automatically if you ask it to.

## Behavior

By default csv-tidy is **strict**:

- every row must have the same number of fields as the first row
- every field must be valid UTF-8
- anything else is a fatal error that names the line number

Pass `--lenient` and it becomes a repair tool instead:

- short rows are padded with empty fields, long rows are truncated
- leading/trailing whitespace is trimmed from every field
- invalid UTF-8 bytes are replaced with `U+FFFD` instead of aborting
- blank lines are dropped

## Usage

```sh
# strict by default - fails fast on a ragged row
$ cat orders.csv
id,name,amount
1,coffee,3.50
2,tea
$ go run . < orders.csv
csv-tidy: line 3: wrong number of fields (use --lenient to pad or truncate rows)

# same file, repaired
$ go run . --lenient < orders.csv
id,name,amount
1,coffee,3.50
2,tea,

# a real file instead of stdin, and a semicolon delimiter
$ go run . --delimiter ";" --lenient export.csv > clean.csv
```

Build a binary the usual way:

```sh
go build -o csv-tidy .
./csv-tidy --lenient messy.csv > clean.csv
```

## Flags

| flag | default | meaning |
|---|---|---|
| `--lenient` | off | repair problems instead of rejecting the file |
| `--delimiter` | `,` | single-character field delimiter |

## Status

Early skeleton. Field-count and UTF-8 checks work; quoting, encoding
detection beyond UTF-8, and header-aware options are not built yet.
