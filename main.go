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
	ID       int
	Title    string
	Director string
	Rating   int
	Favorite bool
	Deleted  bool
}

func createMovieTable(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS movies(
		id INTEGER PRIMARY KEY,
		title TEXT NOT NULL UNIQUE,
		director TEXT NOT NULL,
		rating INTEGER NOT NULL,
		favorite BOOL NOT NULL,
		deleted BOOL DEFAULT false,
		created DATETIME DEFAULT CURRENT_TIMESTAMP);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func queryMovie(db *sql.DB, query string, params []any) (id int, title string, director string, rating int, favorite bool) {
	err := db.QueryRow(query, params...).Scan(&id, &title, &director, &rating, &favorite)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("No movie found with the info of: %s\n", query)
			return 0, "", "", 0, false
		}
		log.Fatal(err)
	}
	return id, title, director, rating, favorite
}

func queryAllDataAllMovies(db *sql.DB) []Movie {
	query := "SELECT id, title, director, rating, favorite FROM movies WHERE DELETED = 0;"
	var id int
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
		err := rows.Scan(&id, &title, &director, &rating, &favorite)
		if err != nil {
			log.Fatal(err)
		}
		data = append(data, Movie{
			ID:       id,
			Title:    title,
			Director: director,
			Rating:   rating,
			Favorite: favorite,
		})
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

func queryMovies(db *sql.DB, query string, params []any) []Movie {
	var id int
	var title string
	var director string
	var rating int
	var favorite bool

	data := []Movie{}
	rows, err := db.Query(query, params...)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &title, &director, &rating, &favorite)
		if err != nil {
			log.Fatal(err)
		}
		data = append(data, Movie{
			Title:    title,
			Director: director,
			Rating:   rating,
			Favorite: favorite,
		})
	}
	return data
}

func deleteFlagMovie(db *sql.DB, query string, params int) int {
	var pk int

	_, err := db.Exec(query, params)
	if err != nil {
		log.Fatal(err)
	}
	return pk
}

func main() {
	// Connect to DB
	db, err := sql.Open("sqlite3", "file:movie-db.db?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		log.Printf("An error has occured: %s\n", err)
	}
	defer db.Close()

	// Creates movie table if not alrady created
	createMovieTable(db)

	home := func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("index.html"))
		tmpl.Execute(w, nil)
	}

	// Inserts movie in DB
	addNewMovie := func(w http.ResponseWriter, r *http.Request) {
		title := r.PostFormValue("title")
		director := r.PostFormValue("director")
		ratingStr := r.PostFormValue("rating")
		ratingInt, err := strconv.Atoi(ratingStr)
		if err != nil {
			http.Error(w, "Invalid integer", http.StatusBadRequest)
		}
		favorite := r.FormValue("favorite") == "true"

		// Insert into new row
		movie := Movie{
			Title:    title,
			Director: director,
			Rating:   ratingInt,
			Favorite: favorite,
		}
		pk := insertMovie(db, movie)
		fmt.Printf("Inserted row ID: %d\n", pk)

		movies := map[string][]Movie{
			"Movies": {
				{
					Title:    title,
					Rating:   ratingInt,
					Director: director,
					Favorite: favorite,
				},
			},
		}
		htmlStr := fmt.Sprintf("%s added", title)
		tmpl, _ := template.New("t").Parse(htmlStr)
		tmpl.Execute(w, movies)
	}

	// Query all rows in DB
	getAll := func(w http.ResponseWriter, r *http.Request) {
		rows := queryAllDataAllMovies(db)

		for _, row := range rows {
			// htmlStr := fmt.Sprintf("<li>%s - %s - %v - %v</li>", row.Title, row.Director, row.Rating, row.Favorite)
			htmlStr := fmt.Sprintf("<li>%s - %s - %v - %v <button hx-delete=\"/delete-movie?id=%d\" hx-target=\"closest li\" hx-swap=\"outerHTML\">🗑️</button></li>", row.Title, row.Director, row.Rating, row.Favorite, row.ID)
			tmpl, _ := template.New("t").Parse(htmlStr)
			tmpl.Execute(w, nil)
		}

		log.Print("HTMX request received")
		log.Print(r.Header.Get("HX-Request"))
	}

	// Build string for DB query and return to user
	getMovie := func(w http.ResponseWriter, r *http.Request) {
		var params []any
		var ratingInt int
		var id int
		query := "SELECT id, title, director, rating, favorite FROM movies WHERE "

		title := r.FormValue("title")
		director := r.FormValue("director")
		ratingStr := r.FormValue("rating")
		// Check if rating is non-empty value and can be converted to int
		if ratingStr != "" {
			converted, err := strconv.Atoi(ratingStr)
			if err != nil {
				http.Error(w, "Invalid integer", http.StatusBadRequest)
				return
			}
			ratingInt = converted
		}
		favorite := r.FormValue("favorite") == "true"

		// If title is provided, QueryRow() is used to return a single result
		if title != "" {
			if len(params) == 0 {
				query += "title LIKE ? "
				params = append(params, "%"+title+"%")
			} else {
				query += "AND title LIKE ? "
				params = append(params, "%"+title+"%")
			}

			id, title, director, ratingInt, favorite = queryMovie(db, query, params)

			// htmlStr := fmt.Sprintf("<li>%s - %s - %v - %v</li>", title, director, ratingInt, favorite)
			htmlStr := fmt.Sprintf("<li>%s - %s - %v - %v <button hx-delete=\"/delete-movie?id=%d\" hx-target=\"closest li\" hx-swap=\"outerHTML\">🗑️</button></li>", title, director, ratingInt, favorite, id)
			tmpl, _ := template.New("t").Parse(htmlStr)
			tmpl.Execute(w, nil)

			log.Print("HTMX request received")
			log.Print(r.Header.Get("HX-Request"))
			return
		}

		// If director, rating or favorite is queried, Query() is used to return multiple results
		if director != "" {
			if len(params) == 0 {
				query += "director LIKE ? "
				params = append(params, "%"+director+"%")
			} else {
				query += "AND director LIKE ? "
				params = append(params, "%"+director+"%")
			}
		}

		if ratingInt != 0 {
			if len(params) == 0 {
				query += "rating = ? "
				params = append(params, ratingInt)
			} else {
				query += "AND rating = ? "
				params = append(params, ratingInt)
			}
		}

		if favorite != false {
			if len(params) == 0 {
				query += "favorite = ? "
				params = append(params, favorite)
			} else {
				query += "AND favorite = ? "
				params = append(params, favorite)
			}
		}

		// fmt.Println(params)
		// fmt.Println(query)

		rows := queryMovies(db, query, params)

		for _, row := range rows {
			// htmlStr := fmt.Sprintf("<li>%s - %s - %v - %v</li>", row.Title, row.Director, row.Rating, row.Favorite)
			htmlStr := fmt.Sprintf("<li>%s - %s - %v - %v <button hx-delete=\"/delete-movie?id=%d\" hx-target=\"closest li\" hx-swap=\"outerHTML\">🗑️</button></li>", row.Title, row.Director, row.Rating, row.Favorite, row.ID)
			tmpl, _ := template.New("t").Parse(htmlStr)
			tmpl.Execute(w, nil)

			log.Print("HTMX request received")
			log.Print(r.Header.Get("HX-Request"))
		}
	}

	deleteMovie := func(w http.ResponseWriter, r *http.Request) {
		responseIdStr := r.URL.Query().Get("id")
		converted, err := strconv.Atoi(responseIdStr)
		if err != nil {
			log.Printf("An error when converting string to int has occured: %s\n", err)
		}
		responseIdInt := converted
		fmt.Printf("Deleted row has ID: %d\n", responseIdInt)
		query := "UPDATE movies SET deleted = 1 WHERE id = ? RETURNING id;"
		pk := deleteFlagMovie(db, query, responseIdInt)
		fmt.Println(pk)
	}

	http.HandleFunc("/", home)
	http.HandleFunc("/add-new-movie", addNewMovie)
	http.HandleFunc("/get-movies", getAll)
	http.HandleFunc("/find-movie", getMovie)
	http.HandleFunc("/delete-movie", deleteMovie)

	log.Fatal(http.ListenAndServe(":8000", nil))
}
