litestream-lib

Forked from litestream([web](https://litestream.io/)|[github](https://github.com/benbjohnson/litestream))

Trying to make it as a lib to init sqlite db client, that can automatically restore and replicate sqlite db to remote, but user can just treat it as golang *sql.DB.

Usage:
- Import lslib package and import your sqlite driver lib(tested cgo-free [modernc sqlite](https://pkg.go.dev/modernc.org/sqlite), and cgo [go-sqlite3](https://github.com/mattn/go-sqlite3)).
```go
import (
    ...

	"github.com/supersupersimple/litestream-lib/lslib"
	_ "modernc.org/sqlite"
)
```
- Setup env to set url and access key and secret
```bash
export LITESTREAM_URL="s3://mybkt.localhost:9000/fruits.db"
export LITESTREAM_ACCESS_KEY_ID=minioadmin
export LITESTREAM_SECRET_ACCESS_KEY=minioadmin
```
- Init and open db.
```go
    lsdb := lslib.NewDB(lslib.NewConfig("test.db").WithDriverName("sqlite"))
	db, err := lsdb.Open(ctx) // db is *sql.DB
	if err != nil {
		panic(err)
	}

	defer lsdb.Close(ctx)
	db.SetMaxOpenConns(1)

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")
```