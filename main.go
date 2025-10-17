package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"

	_ "github.com/go-sql-driver/mysql"
)

// Estructura del usuario
type User struct {
	ID    int
	Name  string
	Email string
	Saldo float64
}

// Datos para la vista
type PageData struct {
	Users   []User
	Error   string
	Success string
	Message string
}

// Estructura de la transferencia
type Transferencia struct {
	ID           int
	EmisorName   string
	ReceptorName string
	Valor        float64
	CreatedAt    string
}

// Variable global para la conexión
var db *sql.DB

func main() {
	var err error
	// Conexión a la base de datos
	// usuario:contraseña@tcp(host:puerto)/base_de_datos
	db, err = sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/banco_go")
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
	http.HandleFunc("/historial", showTransfers)

	fmt.Println("Servidor corriendo en http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

// Muestra los usuarios registrados
func showUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, email, saldo FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Name, &u.Email, &u.Saldo)
		users = append(users, u)
	}

	data := PageData{Users: users}
	// Leer posibles mensajes desde la URL (?success=1&msg=... o ?error=1&msg=...)
	q := r.URL.Query()
	if q.Get("error") != "" {
		data.Error = q.Get("msg")
	}
	if q.Get("success") != "" {
		data.Success = q.Get("msg")
	}
	// Mensaje informativo opcional
	if m := q.Get("info"); m != "" {
		data.Message = m
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Agrega un nuevo usuario
func addUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		name := r.FormValue("name")
		email := r.FormValue("email")

		// Validación simple
		if name == "" || email == "" {
			// Volver a mostrar con error
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

			data := PageData{Users: users, Error: "Nombre y correo son obligatorios"}
			tmpl, err := template.ParseFiles("templates/index.html")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := tmpl.Execute(w, data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		if _, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", name, email); err != nil {
			// Redirigir con error
			target := url.URL{Path: "/", RawQuery: url.Values{"error": {"1"}, "msg": {"No fue posible guardar el usuario"}}.Encode()}
			http.Redirect(w, r, target.String(), http.StatusSeeOther)
			return
		}

		// Redirigir con éxito
		target := url.URL{Path: "/", RawQuery: url.Values{"success": {"1"}, "msg": {"Usuario agregado correctamente"}}.Encode()}
		http.Redirect(w, r, target.String(), http.StatusSeeOther)
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

		// Verificar si el usuario tiene transferencias
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM transferencias WHERE id_emisor = ? OR id_receptor = ?", id, id).Scan(&count)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if count > 0 {
			// Usuario tiene historial de transferencias, no se puede eliminar
			target := url.URL{Path: "/", RawQuery: url.Values{"error": {"1"}, "msg": {"Este usuario no se puede eliminar porque ya tiene un historial de transferencias"}}.Encode()}
			http.Redirect(w, r, target.String(), http.StatusSeeOther)
			return
		}

		// Proceder con la eliminación
		_, err = db.Exec("DELETE FROM users WHERE id = ?", id)
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
	rows, err := db.Query("SELECT id, name FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Name)
		users = append(users, u)
	}

	data := PageData{Users: users}
	tmpl, err := template.ParseFiles("templates/transferencia.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

// Realiza la transferencia
func transferMoney(w http.ResponseWriter, r *http.Request) {
    log.Println("transferMoney called")
    if r.Method != "POST" {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusMethodNotAllowed)
        fmt.Fprintf(w, `{"error": "Método no permitido"}`)
        return
    }

    idEmisor := r.FormValue("id_emisor")
    idReceptor := r.FormValue("id_receptor")
    valor := r.FormValue("valor")

    // Verificar que el emisor tenga saldo suficiente
    var saldoEmisor float64
    err := db.QueryRow("SELECT saldo FROM users WHERE id = ?", idEmisor).Scan(&saldoEmisor)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        fmt.Fprintf(w, `{"error": "Emisor no encontrado"}`)
        return
    }

    valorFloat := 0.0
    fmt.Sscanf(valor, "%f", &valorFloat)

    if saldoEmisor < valorFloat {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        fmt.Fprintf(w, `{"error": "Saldo insuficiente"}`)
        return
    }

    // Verificar que el receptor exista
    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", idReceptor).Scan(&count)
    if err != nil || count == 0 {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        fmt.Fprintf(w, `{"error": "Receptor no encontrado"}`)
        return
    }

    // Iniciar transacción
    tx, err := db.Begin()
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusInternalServerError)
        fmt.Fprintf(w, `{"error": "Error al iniciar la transferencia"}`)
        return
    }
    defer tx.Rollback()

    // Descontar del emisor
    _, err = tx.Exec("UPDATE users SET saldo = saldo - ? WHERE id = ?", valorFloat, idEmisor)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusInternalServerError)
        fmt.Fprintf(w, `{"error": "Error al actualizar saldo del emisor"}`)
        return
    }

    // Sumar al receptor
    _, err = tx.Exec("UPDATE users SET saldo = saldo + ? WHERE id = ?", valorFloat, idReceptor)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusInternalServerError)
        fmt.Fprintf(w, `{"error": "Error al actualizar saldo del receptor"}`)
        return
    }

    // Insertar la transferencia
    _, err = tx.Exec("INSERT INTO transferencias (id_emisor, id_receptor, valor) VALUES (?, ?, ?)", 
        idEmisor, idReceptor, valorFloat)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusInternalServerError)
        fmt.Fprintf(w, `{"error": "Error al registrar la transferencia"}`)
        return
    }

    // Confirmar transacción
    if err = tx.Commit(); err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusInternalServerError)
        fmt.Fprintf(w, `{"error": "Error al realizar la transferencia"}`)
        return
    }

    // Enviar respuesta de éxito
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, `{"message": "Transferencia realizada con éxito"}`)
}
// Muestra todas las transferencias
func showTransfers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT t.id, u1.name AS emisor_name, u2.name AS receptor_name, t.valor, t.created_at
		FROM transferencias t
		JOIN users u1 ON t.id_emisor = u1.id
		JOIN users u2 ON t.id_receptor = u2.id
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var transferencias []Transferencia
	for rows.Next() {
		var t Transferencia
		rows.Scan(&t.ID, &t.EmisorName, &t.ReceptorName, &t.Valor, &t.CreatedAt)
		transferencias = append(transferencias, t)
	}

	tmpl, _ := template.ParseFiles("templates/transferencias.html")
	tmpl.Execute(w, transferencias)
}
