package main

import (
	"context"
	"fmt"
	"log"

	"github.com/supersupersimple/litestream-lib/lslib"
	_ "modernc.org/sqlite"
)

func main() {
	ctx := context.Background()
	lsdb := lslib.NewDB(lslib.NewConfig("test.db").WithDriverName("sqlite"))
	db, err := lsdb.Open(ctx)
	if err != nil {
		panic(err)
	}

	defer db.Close()
	db.SetMaxOpenConns(1)

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")
}
