package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

// Estructura del usuario
type User struct {
	ID    int
	Name  string
	Email string
}

// Variable global para la conexión
var db *sql.DB

func main() {
	var err error
	// Conexión a la base de datos
	// usuario:contraseña@tcp(host:puerto)/base_de_datos
	db, err = sql.Open("mysql", "root:1234@tcp(127.0.0.1:3306)/banco_go")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Rutas
	http.HandleFunc("/", showUsers)
	http.HandleFunc("/add", addUser)
	http.HandleFunc("/edit", editUser)
	http.HandleFunc("/delete", deleteUser)
	http.HandleFunc("/transferencia", showTransferForm)
	http.HandleFunc("/transfer", transferMoney)

	fmt.Println("Servidor corriendo en http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

// Muestra los usuarios registrados
func showUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, email FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Name, &u.Email)
		users = append(users, u)
	}

	tmpl, _ := template.ParseFiles("templates/index.html")
	tmpl.Execute(w, users)
}

// Agrega un nuevo usuario
func addUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		name := r.FormValue("name")
		email := r.FormValue("email")

		_, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", name, email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// Edita un usuario existente
func editUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		id := r.FormValue("id")
		name := r.FormValue("name")
		email := r.FormValue("email")

		_, err := db.Exec("UPDATE users SET name = ?, email = ? WHERE id = ?", name, email, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// Elimina un usuario
func deleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		id := r.URL.Query().Get("id")

		_, err := db.Exec("DELETE FROM users WHERE id = ?", id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// Muestra el formulario de transferencia
func showTransferForm(w http.ResponseWriter, r *http.Request) {
	log.Println("showTransferForm called")
	tmpl, _ := template.ParseFiles("templates/transferencia.html")
	tmpl.Execute(w, nil)
}

// Realiza la transferencia
func transferMoney(w http.ResponseWriter, r *http.Request) {
	log.Println("transferMoney called")
	if r.Method == "POST" {
		log.Println("Method is POST")
		idEmisor := r.FormValue("id_emisor")
		idReceptor := r.FormValue("id_receptor")
		valor := r.FormValue("valor")
		log.Printf("Form values: idEmisor=%s, idReceptor=%s, valor=%s", idEmisor, idReceptor, valor)

		// Insertar la transferencia en la base de datos
		_, err := db.Exec("INSERT INTO transferencias (id_emisor, id_receptor, valor) VALUES (?, ?, ?)", idEmisor, idReceptor, valor)
		if err != nil {
			log.Printf("Database error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println("Transfer inserted successfully")

		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		log.Println("Method is not POST")
	}
}
