// Whimsy APIs
//
// Whimsy API specification
//
// Version: 1.0.0
//
// swagger:meta
package main

import (
	"whimsy/cmd"

	_ "github.com/jackc/pgx/v4/stdlib"
)

func main() {
	cmd.Execute()
}
