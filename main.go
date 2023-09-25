package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/go-sql-driver/mysql"
)

// Declaring a global DB variable to be used throughout the program.
var db *sql.DB

// Album struct represents a row in the "album" table.
type Album struct {
	ID     int64
	Title  string
	Artist string
	Price  float32
}

func main() {
	// Capture connection properties. Using environment variables is a good way
	// to keep sensitive information out of the code.
	cfg := mysql.Config{
		User:                 os.Getenv("DBUSER"), // Fetch database user from environment variable
		Passwd:               os.Getenv("DBPASS"), // Fetch database password from environment variable
		Net:                  "tcp",               // Network type TCP (most common)
		Addr:                 "127.0.0.1:3306",    // Address and port of the database server
		DBName:               "recordings",        // Name of the database
		AllowNativePasswords: true,
	}

	// Attempt to connect to the database using the above configuration.
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err) // If there's an error, it will stop the program.
	}

	// Try to ping the database to ensure connection is alive.
	pingErr := db.Ping() // Verifies if a connection to the database is still alive
	if pingErr != nil {
		log.Fatal(pingErr) // Fatal() is a Print() followed by os.Exit() in case of an error
	}
	fmt.Println("Connected!")

	// Fetch albums with the artist "John Coltrane"
	albums, err := albumsByArtist("John Coltrane")
	if err != nil {
		log.Fatal(err)
	}
	// PrintF() allows the formatting of strings with placeholders ~ in this case %v , aka verb ,is the placeholder
	// \n is an escape sequence that reps a newline character- after the output it ensures a linebreak occurs
	// albums reps what the verb, aka %v, is placeholding
	fmt.Printf("Albums found: %v\n", albums)

	// Fetch the album with ID 2
	alb, err := albumByID(2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Album found: %v\n", alb)

	// Insert a new album into the database and fetch its ID
	albID, err := addAlbum(Album{
		Title:  "The Modern Sound of Betty Carter",
		Artist: "Betty Carter",
		Price:  49.99,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ID of added album: %v\n", albID)
}

// albumsByArtist fetches albums based on the artist's name.
func albumsByArtist(name string) ([]Album, error) {

	// important to note the allocation occuring here is nil
	// since we are unsure of how many structs will be included in the slice - we allow Go to dynamically allocate memory
	// if I knew I wanted 10 slices of type Album struct I would allocate memory immediately to the slice
	// ex ~> albums := make([]Album, 0, 10) translates to create a slice of Album with length 0, aka len(0), and an appendable memory, aka cap(10), of 10 slices
	var albums []Album // declares a variable = to albums that contains slices of type Album struct

	// *************************************
	// Use a prepared SQL query to fetch data.
	// Query() is a provided method from the sql driver pkg and ? is a placeholder for value
	// the provided arg (name) will replace ? after this line has successfully run
	// placeholders are vital in defending against SQL injection attacks
	rows, err := db.Query("SELECT * FROM album WHERE artist = ?", name)

	// Go wants to run error checks after any statements that can provide an error- in this case a DB call
	// this is idiomatic in the Go program and shows up often
	// if this block were to return an error, say a denial of access, it would rep:
	// albumsByArtist "John Coltrane": Error 1045: Access denied for user 'username'@'localhost' (using password: YES)
	if err != nil {
		return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
	}
	// when a db is queried resources are allocated to handle the fetch and result
	// these resources need to be freed after handling their duties
	// so rows.Close() frees these resources after a return result of nil
	// while defer ensures this process occurs after the surrounding func albumsByArtist has finished execution
	// the core of it is a cleanup action
	defer rows.Close() // Ensure resources are freed.

	// *************************************
	// Loop through all fetched rows.
	// rows.Next() will continue through each result from the set returning true until
	// there is no more rows in the result set
	// rows.Next() will return false in that case ending the loop for an exit

	for rows.Next() {
		// var alb Album is declaring a new alb variable for each row in the return row values
		// Isolation is utilized here to ensure that data from one row does not leak into another row
		// so each iteration of this loop starts with a fresh type Album struct assigned to alb
		// Go is a garbage-collected language
		// by redeclaring alb for each iteration, memory from the previous iteration is reclaimed by the garbage collector when no longer needed
		// this becomes of importance in larger database queries
		// Go utilizes 'Zero Value' for undeclared variables
		// int := 0 > float64 := 0.0 > string := "" > bool := false > slice, map, function, pointers, channels := nil
		// arrays := zv.val > structs := zv.field , zv.field
		var alb Album

		// alb.ID := 0 >> alb.Title := "" >> alb.Artist := "" >> alb.Price := 0.0
		// &alb.ID is point telling Scan() exactly where to place found data
		// & is a reference to a variable, in this case alb
		// &alb.ID is a memory referencer for Scan() to accurately place correct data from the current row
		// the arguments of Scan() become pointers and must be placed in exact order from the struct, in this case Album
		// ex ~> &alb.ID := 0 , is a pointer to Scan() for the current row. When that row is scanned (copied) to alb.ID
		// alb.ID := 1
		//so on and so forth
		if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
			return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
		}
		albums = append(albums, alb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
	}
	return albums, nil
}

// albumByID fetches a single album based on its ID.
func albumByID(id int64) (Album, error) {
	var alb Album

	// Fetch only one row based on album ID.
	row := db.QueryRow("SELECT * FROM album WHERE id = ?", id)
	if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
		if err == sql.ErrNoRows { // If no rows are returned, it means no such album exists.
			return alb, fmt.Errorf("albumsById %d: no such album", id)
		}
		return alb, fmt.Errorf("albumsById %d: %v", id, err)
	}
	return alb, nil
}

// addAlbum inserts a new album and returns its ID.
func addAlbum(alb Album) (int64, error) {
	result, err := db.Exec("INSERT INTO album (title, artist, price) VALUES (?, ?, ?)", alb.Title, alb.Artist, alb.Price)
	if err != nil {
		return 0, fmt.Errorf("addAlbum: %v", err)
	}
	id, err := result.LastInsertId() // Fetch the last inserted ID.
	if err != nil {
		return 0, fmt.Errorf("addAlbum: %v", err)
	}
	return id, nil
}
