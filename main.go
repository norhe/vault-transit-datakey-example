package main

import (
  "database/sql"
  "html/template"
  "log"
  "net/http"

  _ "github.com/go-sql-driver/mysql"

)

var db *sql.DB
var err error
var tpl *template.Template


type user_record struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
  Address   string
  Files     []user_file
  Datakey   string
}

type user_file struct {
  ID       int64
  UserID   int64
  Mimetype string
  File     []byte
}

func initDB() {
  dsn := "vault:vaultpw@tcp(127.0.0.1:3306)/my_app"
  db, err = sql.Open("mysql", dsn)

  defer db.Close()

  if err != nil {
    panic(err)
  }

  err = db.Ping()
  if err != nil {
  	log.Fatalln(err)
  }

  create_tables(db)
}

func create_tables(db *sql.DB) {
  _, err = db.Exec("USE my_app")
  if err != nil {
    log.Fatalln(err)
  }

  create_user_table :=
    "CREATE TABLE IF NOT EXISTS `user_data`(" +
    "`user_id` INT(11) NOT NULL AUTO_INCREMENT, " +
    "`user_name` VARCHAR(256) NOT NULL," +
    "`first_name` VARCHAR(256) NULL, " +
    "`last_name` VARCHAR(256) NULL, " +
    "`address` VARCHAR(256) NOT NULL, " +
    "`mime_type` VARCHAR(256) DEFAULT NULL, " +
    "`photo` BLOB DEFAULT NULL, " +
    "`datakey` VARCHAR(256) DEFAULT NULL, " +
    "PRIMARY KEY (user_id) " +
    ") engine=InnoDB;"

  log.Println("Creating user table (if not exist)")

  _, err = db.Exec(create_user_table)
  if err != nil {
    log.Fatalln(err)
  }

  log.Println("Creating files table (if not exist)")

  create_files_table :=
    "CREATE TABLE IF NOT EXISTS `user_files`( " +
    "`file_id` INT(11) NOT NULL AUTO_INCREMENT, " +
    "`user_id` INT(11) NOT NULL, " +
    "`mime_type` VARCHAR(256) DEFAULT NULL, " +
    "`file` BLOB DEFAULT NULL, " +
    "PRIMARY KEY (file_id) " +
    ") engine=InnoDB;"

    _, err = db.Exec(create_files_table)
    if err != nil {
      log.Fatalln(err)
    }
}

func initTemplates() {
  tpl = template.Must(template.ParseGlob("templates/*"))
}

func main() {
  //defer db.Close()
  initDB()
  initTemplates()

  url := "0.0.0.0:1234"

  // set up routes
  http.Handle("/favicon.ico", http.NotFoundHandler())
  http.HandleFunc("/", indexHandler)
  http.HandleFunc("/view", viewHandler)
  http.HandleFunc("/create", createHandler)
  http.HandleFunc("/update", updateHandler)

  // run the server
  log.Printf("Server is running at http://%s", url)
  log.Fatalln(http.ListenAndServe(url, nil))
}

func indexHandler(w http.ResponseWriter, req *http.Request) {
  tpl.ExecuteTemplate(w, "index.html", nil)
}

func createHandler(w http.ResponseWriter, req *http.Request) {
  tpl.ExecuteTemplate(w, "create.html", nil)
}

func viewHandler(w http.ResponseWriter, req *http.Request) {
  tpl.ExecuteTemplate(w, "view.html", nil)
}

func updateHandler(w http.ResponseWriter, req *http.Request) {
  tpl.ExecuteTemplate(w, "update.html", nil)
}
