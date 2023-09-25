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
	// ex ~> albums := make([]Album, 0, 10) translates to create a slice of Album with len(0) and an appendable memory, aka cap(10), of 10 slices
	var albums []Album // declares a variable = to albums that contains slices of type Album struct

	// Use a prepared SQL query to fetch data.
	rows, err := db.Query("SELECT * FROM album WHERE artist = ?", name)
	if err != nil {
		return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
	}
	defer rows.Close() // Ensure resources are freed.

	// Loop through all fetched rows.
	for rows.Next() {
		var alb Album
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
