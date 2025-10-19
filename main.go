package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

type Movie struct {
	Title    string
	Director string
	Rating   int8
	Favorite bool
}

func createMovieTable(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS movies(
		id INTEGER PRIMARY KEY,
		title TEXT NOT NULL,
		director TEXT NOT NULL,
		rating INTEGER NOT NULL,
		favorite BOOL NOT NULL,
		created DATETIME DEFAULT CURRENT_TIMESTAMP);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func findMovie(db *sql.DB) (title string, director string, rating int8, favorite bool) {
	query := "SELECT title, director, rating, favorite FROM movies WHERE title = 'The Batman';"

	err := db.QueryRow(query).Scan(&title, &director, &rating, &favorite)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Fatalf("No movie found with the name of: %s\n", "The Fartman")
		}
		log.Fatal(err)
	}
	return title, director, rating, favorite
}

func findAllDataAllMovies(db *sql.DB) []Movie {
	query := "SELECT title, director, rating, favorite FROM movies;"

	var title string
	var director string
	var rating int8
	var favorite bool

	data := []Movie{}
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&title, &director, &rating, &favorite)
		if err != nil {
			log.Fatal(err)
		}
		data = append(data, Movie{title, director, rating, favorite})
	}

	return data
}

func insertMovie(db *sql.DB, movie Movie) int {
	query := `INSERT INTO movies (title, director, rating, favorite)
		VALUES (?, ?, ?, ?) RETURNING id;`

	var pk int

	err := db.QueryRow(query, movie.Title, movie.Director, movie.Rating, movie.Favorite).Scan(&pk)
	if err != nil {
		log.Fatal(err)
	}
	return pk
}

func main() {
	// Connect to DB
	db, _ := sql.Open("sqlite3", "movie-db.db")
	defer db.Close()
	db.Exec(`PRAGMA journal_mode=WAL`)

	// Creates movie table if not alrady created
	createMovieTable(db)

	// Inserts movie in DB
	movie := Movie{"American Psycho", "Mary Harron", 7, false}
	pk := insertMovie(db, movie)
	fmt.Printf("Inserted row ID: %d\n", pk)

	// Query DB for movie
	title, director, rating, favorite := findMovie(db)
	fmt.Printf("Found: %s, %s, %d, %t\n", title, director, rating, favorite)

	// Query all rows in DB
	rows := findAllDataAllMovies(db)
	fmt.Printf("All rows: %v\n", rows)

	h1 := func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("index.html"))
		movies := map[string][]Movie{
			"Movies": {
				{Title: "The Interview", Rating: 9, Director: "Seth Rogen"},
			},
		}
		tmpl.Execute(w, movies)
	}

	h2 := func(w http.ResponseWriter, r *http.Request) {
		title := r.PostFormValue("title")
		director := r.PostFormValue("director")
		rating := r.PostFormValue("rating")
		htmlStr := fmt.Sprintf("<li>%s - %s - %v</li>", title, director, rating)
		tmpl, _ := template.New("t").Parse(htmlStr)
		tmpl.Execute(w, nil)

		log.Print("HTMX request received")
		log.Print(r.Header.Get("HX-Request"))
	}

	http.HandleFunc("/", h1)
	http.HandleFunc("/add-movie", h2)

	log.Fatal(http.ListenAndServe(":8000", nil))
}
