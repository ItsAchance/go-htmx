package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

type Movie struct {
	Title    string
	Director string
	Rating   int
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

func findMovie(db *sql.DB) (title string, director string, rating int, favorite bool) {
	query := "SELECT title, director, rating, favorite FROM movies WHERE title = ?;"

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
	var rating int
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

	home := func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("index.html"))
		tmpl.Execute(w, nil)
	}

	addNewMovie := func(w http.ResponseWriter, r *http.Request) {
		// Inserts movie in DB
		title := r.PostFormValue("title")
		director := r.PostFormValue("director")
		ratingStr := r.PostFormValue("rating")
		ratingInt, err := strconv.Atoi(ratingStr)
		if err != nil {
			http.Error(w, "Invalid integer", http.StatusBadRequest)
		}
		favorite := r.FormValue("favorite") == "true"

		movie := Movie{title, director, ratingInt, favorite}
		pk := insertMovie(db, movie)
		fmt.Printf("Inserted row ID: %d\n", pk)
		tmpl := template.Must(template.ParseFiles("index.html"))

		movies := map[string][]Movie{
			"Movies": {
				{Title: title, Rating: ratingInt, Director: director, Favorite: favorite},
			},
		}
		tmpl.Execute(w, movies)
	}

	getAll := func(w http.ResponseWriter, r *http.Request) {
		// Query all rows in DB
		rows := findAllDataAllMovies(db)
		fmt.Printf("All rows: %v\n", rows)

		// htmlStr := fmt.Sprintf("<li>%s - %s - %v - %v</li>", title, director, rating, favorite)
		// tmpl, _ := template.New("t").Parse(htmlStr)
		// tmpl.Execute(w, nil)

		log.Print("HTMX request received")
		log.Print(r.Header.Get("HX-Request"))
	}

	findMovie := func(w http.ResponseWriter, r *http.Request) {
		// Query DB for movie
		inputTitle := r.PostFormValue("new-title")
		fmt.Println(inputTitle)
		inputDirector := r.PostFormValue("director")
		inputRating := r.PostFormValue("rating")
		inputFavorite := r.PostFormValue("favorite")
		fmt.Println(inputDirector)
		fmt.Println(inputRating)
		fmt.Println(inputFavorite)
		// inputrows := findMovie(db)

		htmlStr := fmt.Sprintf("<li>%s - %s - %v - %v</li>", inputTitle, inputDirector, inputRating, inputFavorite)
		tmpl, _ := template.New("t").Parse(htmlStr)
		tmpl.Execute(w, nil)

		log.Print("HTMX request received")
		log.Print(r.Header.Get("HX-Request"))
	}

	http.HandleFunc("/", home)
	http.HandleFunc("/add-new-movie", addNewMovie)
	http.HandleFunc("/get-movies", getAll)
	http.HandleFunc("/find-movie", findMovie)

	log.Fatal(http.ListenAndServe(":8000", nil))
}
