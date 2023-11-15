package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	Username string
	Password string
}

var sessionMan *scs.SessionManager

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	msg := sessionMan.PopString(r.Context(), "flash")
	t := template.Must(template.ParseFiles("templates/login.html"))
	t.Execute(w, msg)
}

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("templates/profile.html"))
	username := sessionMan.GetString(r.Context(), "username")
	fmt.Println(username)
	t.Execute(w, username)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("templates/index.html"))
	t.Execute(w, nil)
}

func SigninHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	if !(username == "robson" && password == "123456") {
		//flash message
		sessionMan.Put(r.Context(), "flash", "invalid credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	sessionMan.Put(r.Context(), "username", username)

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func SignoutHandler(w http.ResponseWriter, r *http.Request) {
	sessionMan.Destroy(r.Context())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func SecureMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := sessionMan.GetString(r.Context(), "username")
		if len(username) == 0 {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func createStoreSessionTable(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			expiry REAL NOT NULL
		)
	`)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expiry)")
	if err != nil {
		panic(err)
	}
}

func main() {
	db, err := sql.Open("sqlite3", "sessions.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	createStoreSessionTable(db)

	sessionMan = scs.New()
	sessionMan.Lifetime = time.Second * 10
	sessionMan.Store = sqlite3store.New(db)

	sqlite3store.NewWithCleanupInterval(db, time.Second*30)

	mux := http.NewServeMux()

	mux.HandleFunc("/", IndexHandler)
	mux.HandleFunc("/login", LoginHandler)
	mux.HandleFunc("/profile", SecureMiddleware(ProfileHandler))

	mux.HandleFunc("/signin", SigninHandler)
	mux.HandleFunc("/signout", SignoutHandler)

	http.ListenAndServe(":3000", sessionMan.LoadAndSave(mux))
}
