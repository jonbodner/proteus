module github.com/jonbodner/proteus

go 1.25.0

require (
	github.com/go-sql-driver/mysql v1.5.0
	github.com/google/go-cmp v0.6.0
	github.com/jonbodner/dbtimer v0.0.0-20170410163237-7002f3758ae1
	github.com/lib/pq v1.10.9
	github.com/pkg/profile v1.7.0
	github.com/rickar/props v1.0.0
)

require (
	github.com/BurntSushi/toml v1.4.1-0.20240526193622-a339e1f7089c // indirect
	github.com/felixge/fgprof v0.9.3 // indirect
	github.com/google/pprof v0.0.0-20230429030804-905365eefe3e // indirect
	golang.org/x/exp/typeparams v0.0.0-20231108232855-2478ac86f678 // indirect
	golang.org/x/mod v0.31.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/tools v0.40.1-0.20260108161641-ca281cf95054 // indirect
	honnef.co/go/tools v0.7.0 // indirect
)

tool honnef.co/go/tools/cmd/staticcheck
