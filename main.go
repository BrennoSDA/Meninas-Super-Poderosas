package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	// Inicializar conexão com o banco de dados
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Falha ao conectar ao banco de dados: %v", err)
	}
	defer db.Close()

	// Configurar sessionStore
	sessionStore = sessions.NewCookieStore([]byte("sua-chave-secreta-segura"))
	sessionStore.Options = &sessions.Options{
		MaxAge:   86400 * 7, // 1 semana
		HttpOnly: true,
	}

	// Configurar rotas
	router := http.NewServeMux()

	// Rotas estáticas
	router.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Rotas de autenticação
	router.HandleFunc("/", loginHandler)
	router.HandleFunc("/login", loginHandler)
	router.HandleFunc("/logout", logoutHandler)
	router.HandleFunc("/registro", registroHandler)

	// Rotas do profissional
	router.HandleFunc("/profissional/inicio", profissionalInicioHandler)
	router.HandleFunc("/profissional/cadastrar-paciente", cadastrarPacienteHandler)
	router.HandleFunc("/profissional/buscar-paciente", buscarPacienteHandler)
	router.HandleFunc("/profissional/cadastrar-exame", cadastrarExameHandler)

	// Rotas de exames
	router.HandleFunc("/exames/anamnese", anamneseHandler)
	router.HandleFunc("/exames/laboratorio", laboratorioHandler)
	router.HandleFunc("/exames/resultado", resultadoHandler)

	// Rotas do paciente
	router.HandleFunc("/paciente/inicio", pacienteInicioHandler)
	router.HandleFunc("/paciente/marcar-consulta", marcarConsultaHandler)
	router.HandleFunc("/paciente/ver-resultados", verResultadosHandler)
	router.HandleFunc("/paciente/resultado-detalhes", resultadoPacienteHandler)

	// Configurar servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	log.Printf("Servidor rodando na porta %s\n", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}

