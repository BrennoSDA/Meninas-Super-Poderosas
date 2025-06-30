package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/lib/pq"
)

var sessionStore *sessions.CookieStore

// Handlers de autenticação
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		http.ServeFile(w, r, "static/Login/Login.html")
		return
	}

	// Processar login
	email := r.FormValue("email")
	senha := r.FormValue("senha")

	var usuario Usuario
	err := db.QueryRow("SELECT id, email, tipo FROM usuarios WHERE email = $1 AND senha = $2",
		email, senha).Scan(&usuario.ID, &usuario.Email, &usuario.Tipo)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Credenciais inválidas", http.StatusUnauthorized)
		} else {
			http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		}
		return
	}

	// Criar sessão (simplificado - futuramente usar sessões seguras)
	if usuario.Tipo == "profissional" {
		http.Redirect(w, r, "/profissional/inicio", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/paciente/inicio", http.StatusSeeOther)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Limpar sessão (simplificado)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func registroHandler(w http.ResponseWriter, r *http.Request) {
	// JSON contendo o tipo
	w.Header().Set("Content-Type", "application/json")

	// base responsiva struct
	response := map[string]interface{}{
		"success": false,
		"message": "",
		"id":      0,
		"type":    "",
	}

	// Debug: registra o método de solicitação e o tipo de conteúdo
	log.Printf("[REGISTER] Method: %s, Content-Type: %s", r.Method, r.Header.Get("Content-Type"))

	if r.Method == "GET" {
		http.ServeFile(w, r, "static/Registro/Registro.html")
		return
	}

	// Analisa o formulário por multipartes (para uploads de arquivos)
	// Analise do formulário regular
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		if err := r.ParseForm(); err != nil {
			log.Printf("[REGISTER] Failed to parse form: %v", err)
			response["message"] = "Invalid form data"
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// envia todas os valores
	formValues := r.Form
	log.Printf("[REGISTER] Form values received: %+v", formValues)

	// Extração de dados
	email := strings.TrimSpace(formValues.Get("email"))
	senha := formValues.Get("senha")
	confirmarSenha := formValues.Get("confirmarSenha")
	tipo := formValues.Get("tipo")

	// Validação de dados obrigatórios
	if email == "" {
		response["message"] = "Email is required"
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if senha == "" {
		response["message"] = "Password is required"
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if senha != confirmarSenha {
		response["message"] = "Passwords don't match"
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Processo de registro baseado no tipo
	var id int
	var err error

	switch tipo {
	case "profissional":
		cnes := strings.TrimSpace(formValues.Get("cnes"))
		if cnes == "" {
			response["message"] = "CNES is required for professionals"
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		log.Printf("[REGISTER] Registering professional: %s, CNES: %s", email, cnes)

		err = db.QueryRow(`
            INSERT INTO usuarios (email, senha, tipo, cnes, created_at, updated_at)
            VALUES ($1, $2, $3, $4, NOW(), NOW())
            RETURNING id`,
			email, senha, tipo, cnes).Scan(&id)

	case "paciente":
		cpf := strings.ReplaceAll(strings.TrimSpace(formValues.Get("cpf")), ".", "")
		cpf = strings.ReplaceAll(cpf, "-", "")
		telefone := strings.TrimSpace(formValues.Get("telefone"))

		if len(cpf) != 11 {
			response["message"] = "CPF must have 11 digits"
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		log.Printf("[REGISTER] Registering patient: %s, CPF: %s", email, cpf)

		err = db.QueryRow(`
            INSERT INTO usuarios (email, senha, tipo, cpf, telefone, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
            RETURNING id`,
			email, senha, tipo, cpf, telefone).Scan(&id)

	default:
		response["message"] = "Invalid user type"
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Handle database erros
	if err != nil {
		log.Printf("[REGISTER] Database error: %v", err)

		if pqErr, ok := err.(*pq.Error); ok {
			log.Printf("[REGISTER] PostgreSQL error: %+v", pqErr)

			// Handle unique violation (duplicate email)
			if pqErr.Code == "23505" {
				response["message"] = "Email already registered"
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(response)
				return
			}
		}

		response["message"] = "Registration failed"
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// registro com sucesso
	log.Printf("[REGISTER] Successfully registered user ID: %d", id)

	response["success"] = true
	response["id"] = id
	response["type"] = tipo
	response["message"] = "Registration successful"

	json.NewEncoder(w).Encode(response)
}

// Handlers do profissional
func profissionalInicioHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/Profissional/inicioProfissional.html")
}

func cadastrarPacienteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		http.ServeFile(w, r, "static/Cadastro/Cadastro.html")
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Erro ao processar formulário", http.StatusBadRequest)
		return
	}

	// Converter data
	dataNasc, err := time.Parse("2006-01-02", r.FormValue("data_nascimento"))
	if err != nil {
		http.Error(w, "Formato de data inválido (use DD/MM/AAAA)", http.StatusBadRequest)
		return
	}

	paciente := Paciente{
		// Dados da Unidade de Saúde
		UF:               r.FormValue("uf"),
		CNES:             r.FormValue("cnes"),
		Protocolo:        toNullString(r.FormValue("protocolo")),
		NomeUnidade:      r.FormValue("nome_unidade"),
		MunicipioUnidade: r.FormValue("municipio"),

		// Informações Pessoais
		CartaoSUS:      r.FormValue("cartao_sus"),
		NomeCompleto:   r.FormValue("nome_completo"),
		NomeMae:        r.FormValue("nome_mae"),
		CPF:            toNullString(r.FormValue("cpf")),
		Apelido:        toNullString(r.FormValue("apelido")),
		DataNascimento: dataNasc,
		Etnia:          toNullString(r.FormValue("etnia")),
		EtniaOutra:     toNullString(r.FormValue("etnia_outra")),

		// Endereço
		Logradouro:      toNullString(r.FormValue("logradouro")),
		Numero:          toNullString(r.FormValue("numero")),
		Complemento:     toNullString(r.FormValue("complemento")),
		Bairro:          toNullString(r.FormValue("bairro")),
		UFEndereco:      toNullString(r.FormValue("uf_endereco")),
		CodigoMunicipio: toNullString(r.FormValue("codigo_municipio")),
		Municipio:       toNullString(r.FormValue("municipio_endereco")),
		CEP:             toNullString(r.FormValue("cep")),
		DDD:             toNullString(r.FormValue("ddd")),
		Telefone:        toNullString(r.FormValue("telefone")),
		PontoReferencia: toNullString(r.FormValue("ponto_referencia")),
		Escolaridade:    toNullString(r.FormValue("escolaridade")),
	}

	// Validar campos obrigatórios
	if paciente.CartaoSUS == "" || paciente.NomeCompleto == "" || paciente.NomeMae == "" {
		http.Error(w, "Campos obrigatórios faltando", http.StatusBadRequest)
		return
	}

	// Inserir no banco
	_, err = db.Exec(`
        INSERT INTO pacientes (
            uf, cnes, protocolo, nome_unidade, municipio_unidade,
            cartao_sus, nome_completo, nome_mae, cpf, apelido,
            data_nascimento, etnia, etnia_outra,
            logradouro, numero, complemento, bairro,
            uf_endereco, codigo_municipio, municipio_endereco,
            cep, ddd, telefone, ponto_referencia, escolaridade
        ) VALUES (
            $1, $2, $3, $4, $5,
            $6, $7, $8, $9, $10,
            $11, $12, $13,
            $14, $15, $16, $17,
            $18, $19, $20,
            $21, $22, $23, $24, $25
        )`,
		// Valores na mesma ordem
		paciente.UF, paciente.CNES, paciente.Protocolo, paciente.NomeUnidade, paciente.MunicipioUnidade,
		paciente.CartaoSUS, paciente.NomeCompleto, paciente.NomeMae, paciente.CPF, paciente.Apelido,
		paciente.DataNascimento, paciente.Etnia, paciente.EtniaOutra,
		paciente.Logradouro, paciente.Numero, paciente.Complemento, paciente.Bairro,
		paciente.UFEndereco, paciente.CodigoMunicipio, paciente.Municipio,
		paciente.CEP, paciente.DDD, paciente.Telefone, paciente.PontoReferencia, paciente.Escolaridade,
	)

	if err != nil {
		log.Printf("Erro ao cadastrar paciente: %v", err)
		http.Error(w, "Erro ao salvar paciente", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profissional/inicio?success=1", http.StatusSeeOther)
}

// Função auxiliar para campos opcionais
func toNullString(s string) sql.NullString {
	s = strings.TrimSpace(s)
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

func buscarPacienteHandler(w http.ResponseWriter, r *http.Request) {
	// Permitir tanto GET quanto POST
	var cpf string
	if r.Method == "GET" {
		cpf = r.URL.Query().Get("cpf")
	} else {
		cpf = r.FormValue("cpf")
	}

	// Remover toda formatação do CPF
	cpf = strings.ReplaceAll(cpf, ".", "")
	cpf = strings.ReplaceAll(cpf, "-", "")
	cpf = strings.ReplaceAll(cpf, " ", "")

	if len(cpf) != 11 {
		http.Error(w, "CPF deve conter 11 dígitos", http.StatusBadRequest)
		return
	}

	var paciente struct {
		ID           int    `json:"id"`
		NomeCompleto string `json:"nome_completo"`
		CPF          string `json:"cpf"`
	}

	// Buscar paciente no banco (considerando CPF sem formatação)
	query := `SELECT id, nome_completo, cpf FROM pacientes WHERE REPLACE(REPLACE(cpf, '.', ''), '-', '') = $1`
	err := db.QueryRow(query, cpf).Scan(&paciente.ID, &paciente.NomeCompleto, &paciente.CPF)

	if err != nil {
		if err == sql.ErrNoRows {
			// Tentar buscar por outros critérios se necessário
			http.Error(w, "Paciente não encontrado no sistema", http.StatusNotFound)
		} else {
			log.Printf("Erro ao buscar paciente: %v", err)
			http.Error(w, "Erro interno ao buscar paciente", http.StatusInternalServerError)
		}
		return
	}

	// Retornar resposta JSON com dados formatados
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(paciente); err != nil {
		log.Printf("Erro ao codificar resposta: %v", err)
		http.Error(w, "Erro ao processar resposta", http.StatusInternalServerError)
	}
}

func cadastrarExameHandler(w http.ResponseWriter, r *http.Request) {
	pacienteID := r.FormValue("paciente_id")
	if pacienteID == "" {
		// Tentar obter do query parameter para compatibilidade
		pacienteID = r.URL.Query().Get("paciente_id")
		if pacienteID == "" {
			http.Error(w, "ID do paciente não fornecido", http.StatusBadRequest)
			return
		}
	}

	pid, err := strconv.Atoi(pacienteID)
	if err != nil {
		http.Error(w, "ID do paciente inválido", http.StatusBadRequest)
		return
	}

	// Determinar qual exame está sendo cadastrado
	tipoExame := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

	switch tipoExame {
	case "anamnese":
		processarAnamnese(w, r, pid)
	case "laboratorio":
		processarLaboratorio(w, r, pid)
	case "resultado":
		processarResultado(w, r, pid)
	default:
		http.Error(w, "Tipo de exame inválido", http.StatusBadRequest)
	}
}

// Handlers de exames
func anamneseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		pacienteID := r.URL.Query().Get("paciente_id")
		if pacienteID == "" {
			// Tentar obter do localStorage via template
			http.ServeFile(w, r, "static/Anamnese/Anamnese.html")
			return
		}

		// Verificar se o paciente existe
		var pacienteExists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM pacientes WHERE id = $1)", pacienteID).Scan(&pacienteExists)
		if err != nil || !pacienteExists {
			http.Error(w, "Paciente não encontrado", http.StatusNotFound)
			return
		}

		http.ServeFile(w, r, "static/Anamnese/Anamnese.html")
		return
	}

	// Processar POST
	pacienteID := r.FormValue("paciente_id")
	if pacienteID == "" {
		// Tentar obter do localStorage via AJAX
		http.Error(w, "ID do paciente não fornecido", http.StatusBadRequest)
		return
	}

	pid, err := strconv.Atoi(pacienteID)
	if err != nil {
		http.Error(w, "ID do paciente inválido", http.StatusBadRequest)
		return
	}

	processarAnamnese(w, r, pid)
}

func laboratorioHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		pacienteID := r.URL.Query().Get("paciente_id")
		if pacienteID == "" {
			http.Error(w, "ID do paciente é obrigatório", http.StatusBadRequest)
			return
		}

		// Verificar se o paciente existe
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM pacientes WHERE id = $1)", pacienteID).Scan(&exists)
		if err != nil || !exists {
			http.Error(w, "Paciente não encontrado", http.StatusNotFound)
			return
		}

		http.ServeFile(w, r, "static/Laboratório/Laboratório.html")
		return
	}

	// Processar POST
	pacienteID := r.FormValue("paciente_id")
	if pacienteID == "" {
		http.Error(w, "ID do paciente é obrigatório", http.StatusBadRequest)
		return
	}

	pid, err := strconv.Atoi(pacienteID)
	if err != nil {
		http.Error(w, "ID do paciente inválido", http.StatusBadRequest)
		return
	}

	processarLaboratorio(w, r, pid)
}

func processarLaboratorio(w http.ResponseWriter, r *http.Request, pacienteID int) {
	// 1. Processar dados do laboratório
	cnesLab := r.FormValue("cnes_lab")
	nomeLab := r.FormValue("nome_lab")

	// 2. Inserir ou obter ID do laboratório
	var laboratorioID int
	err := db.QueryRow(`
        INSERT INTO laboratorios (cnes, nome_laboratorio) 
        VALUES ($1, $2)
        ON CONFLICT (cnes) DO UPDATE 
        SET nome_laboratorio = EXCLUDED.nome_laboratorio
        RETURNING id`,
		cnesLab, nomeLab).Scan(&laboratorioID)

	if err != nil {
		log.Printf("Erro ao salvar laboratório: %v", err)
		http.Error(w, "Erro ao cadastrar laboratório", http.StatusInternalServerError)
		return
	}

	// 3. Processar exame com o laboratorio_id obtido
	exame := ExameCitopatologico{
		PacienteID:             pacienteID,
		LaboratorioID:          laboratorioID,
		NumeroExame:            r.FormValue("numero_exame"),
		RecebidoEm:             parseDate(r.FormValue("recebido_em")),
		AmostraRejeitada:       sql.NullString{String: r.FormValue("amostra_rejeitada"), Valid: r.FormValue("amostra_rejeitada") != ""},
		CausaRejeicao:          sql.NullString{String: r.FormValue("causa_rejeicao"), Valid: r.FormValue("causa_rejeicao") != ""},
		Epitelios:              r.Form["epitelio"],
		AdequabilidadeMaterial: sql.NullString{String: r.FormValue("adequabilidade_material"), Valid: r.FormValue("adequabilidade_material") != ""},
		MotivoInsatisfatorio:   sql.NullString{String: strings.Join(r.Form["motivo_insatisfatorio"], ", "), Valid: len(r.Form["motivo_insatisfatorio"]) > 0},
	}

	// 4. Inserir exame usando TODOS os campos da struct
	_, err = db.Exec(`
        INSERT INTO exames_citopatologicos (
            paciente_id, laboratorio_id, numero_exame, recebido_em,
            amostra_rejeitada, causa_rejeicao, epitelios,
            adequabilidade_material, motivo_insatisfatorio
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		exame.PacienteID,
		exame.LaboratorioID,
		exame.NumeroExame,
		exame.RecebidoEm,
		exame.AmostraRejeitada,
		exame.CausaRejeicao,
		pq.Array(exame.Epitelios),
		exame.AdequabilidadeMaterial,
		exame.MotivoInsatisfatorio,
	)

	if err != nil {
		log.Printf("Erro ao salvar exame: %v", err)
		http.Error(w, "Erro ao salvar exame", http.StatusInternalServerError)
		return
	}

	// Em processarAnamnese, processarLaboratorio e processarResultado
	http.Redirect(w, r, "/static/Cadastrar exames/cadastroExame.html", http.StatusSeeOther)
}
func resultadoHandler(w http.ResponseWriter, r *http.Request) {
	// Tratamento para GET
	if r.Method == "GET" {
		pacienteID := r.URL.Query().Get("paciente_id")
		if pacienteID == "" {
			http.Error(w, "ID do paciente é obrigatório", http.StatusBadRequest)
			return
		}

		// Verifica se o paciente existe
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM pacientes WHERE id = $1)", pacienteID).Scan(&exists)
		if err != nil || !exists {
			http.Error(w, "Paciente não encontrado", http.StatusNotFound)
			return
		}

		http.ServeFile(w, r, "static/Resultado/Resultado.html")
		return
	}

	// Tratamento para POST
	pacienteID := r.FormValue("paciente_id")
	if pacienteID == "" {
		// Tenta obter do corpo JSON (caso seja uma API)
		var data struct {
			PacienteID string `json:"paciente_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&data); err == nil {
			pacienteID = data.PacienteID
		}

		if pacienteID == "" {
			http.Error(w, "ID do paciente é obrigatório", http.StatusBadRequest)
			return
		}
	}

	pid, err := strconv.Atoi(pacienteID)
	if err != nil {
		http.Error(w, "ID do paciente inválido", http.StatusBadRequest)
		return
	}

	processarResultado(w, r, pid)
}

func processarResultado(w http.ResponseWriter, r *http.Request, pacienteID int) {
	// Primeiro, obter o ID do último exame do paciente
	var exameID int
	err := db.QueryRow(`
        SELECT id FROM exames_citopatologicos 
        WHERE paciente_id = $1 
        ORDER BY created_at DESC 
        LIMIT 1
    `, pacienteID).Scan(&exameID)

	if err != nil {
		http.Error(w, "Nenhum exame encontrado para este paciente", http.StatusBadRequest)
		return
	}

	// Processar formulário
	diagnostico := Diagnostico{
		ExameID:                  exameID,
		DentroLimitesNormais:     sql.NullBool{Bool: r.FormValue("dentro_limites") == "sim", Valid: true},
		AlteracoesBenignas:       r.Form["alteracoes_benignas"],
		OutrasAlteracoesBenignas: sql.NullString{String: r.FormValue("outras_alteracoes"), Valid: r.FormValue("outras_alteracoes") != ""},
		CelulasAtipicas:          sql.NullString{String: r.FormValue("celulas_atipicas"), Valid: r.FormValue("celulas_atipicas") != ""},
		AtipiasCelulares:         r.Form["atipias_celulares"],
		OutrasNeoplasias:         sql.NullString{String: r.FormValue("outras_neoplasias"), Valid: r.FormValue("outras_neoplasias") != ""},
		CelulasEndometriais:      sql.NullBool{Bool: r.FormValue("celulas_endometriais") == "on", Valid: true},
		ObservacoesGerais:        sql.NullString{String: r.FormValue("observacoes_gerais"), Valid: r.FormValue("observacoes_gerais") != ""},
		ScreeningCitotecnico:     sql.NullString{String: r.FormValue("screening_citotecnico"), Valid: r.FormValue("screening_citotecnico") != ""},
		Responsavel:              r.FormValue("responsavel"),
		DataResultado:            parseDate(r.FormValue("data_resultado")),
	}

	// Query de inserção
	_, err = db.Exec(`
        INSERT INTO diagnosticos (
            exame_id, dentro_limites_normais, alteracoes_benignas, outras_alteracoes_benignas,
            celulas_atipicas, atipias_celulares, outras_neoplasias, celulas_endometriais,
            observacoes_gerais, screening_citotecnico, responsavel, data_resultado
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `,
		diagnostico.ExameID,
		diagnostico.DentroLimitesNormais,
		pq.Array(diagnostico.AlteracoesBenignas),
		diagnostico.OutrasAlteracoesBenignas,
		diagnostico.CelulasAtipicas,
		pq.Array(diagnostico.AtipiasCelulares),
		diagnostico.OutrasNeoplasias,
		diagnostico.CelulasEndometriais,
		diagnostico.ObservacoesGerais,
		diagnostico.ScreeningCitotecnico,
		diagnostico.Responsavel,
		diagnostico.DataResultado,
	)

	if err != nil {
		log.Printf("Erro ao salvar diagnóstico: %v", err)
		http.Error(w, "Erro ao salvar diagnóstico", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/static/Cadastrar exames/cadastroExame.html", http.StatusSeeOther)
}

// Handlers do paciente
func pacienteInicioHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/Paciente/inicioPaciente.html")
}

func marcarConsultaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		http.ServeFile(w, r, "static/Paciente/inicioPaciente.html") // Ou crie uma página específica
		return
	}

	// Processar marcação de consulta
	consulta := Consulta{
		PacienteID:    1, // TODO: obter da sessão
		Especialidade: r.FormValue("especialidade"),
		DataConsulta:  parseDate(r.FormValue("data_consulta")),
		Horario:       parseTime(r.FormValue("horario")),
		LocalConsulta: "Unidade Básica de Saúde", // Pode ser dinâmico
	}

	_, err := db.Exec(`INSERT INTO consultas (
        paciente_id, especialidade, data_consulta, horario, local_consulta
    ) VALUES ($1, $2, $3, $4, $5)`,
		consulta.PacienteID, consulta.Especialidade, consulta.DataConsulta,
		consulta.Horario, consulta.LocalConsulta)
	if err != nil {
		http.Error(w, "Erro ao marcar consulta", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/paciente/inicio", http.StatusSeeOther)
}

func verResultadosHandler(w http.ResponseWriter, r *http.Request) {
	pacienteID := 1 // TODO: obter da sessão

	var resultados []struct {
		Tipo      string
		Data      time.Time
		Resultado string
		Link      string
	}

	rows, err := db.Query(`
        SELECT 'Citopatológico' as tipo, data_resultado as data, 
               CASE WHEN dentro_limites_normais THEN 'Normal' ELSE 'Alterado' END as resultado,
               '/exames/resultado/' || id as link
        FROM diagnosticos
        WHERE exame_id IN (SELECT id FROM exames_citopatologicos WHERE paciente_id = $1)
        UNION ALL
        SELECT 'Clínico' as tipo, data_coleta as data, 
               CASE WHEN inspecao_colo = 'Normal' THEN 'Normal' ELSE 'Alterado' END as resultado,
               '/exames/clinico/' || id as link
        FROM exames_clinicos
        WHERE paciente_id = $1
    `, pacienteID)

	if err != nil {
		http.Error(w, "Erro ao buscar resultados", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var res struct {
			Tipo      string
			Data      time.Time
			Resultado string
			Link      string
		}
		if err := rows.Scan(&res.Tipo, &res.Data, &res.Resultado, &res.Link); err != nil {
			http.Error(w, "Erro ao ler resultados", http.StatusInternalServerError)
			return
		}
		resultados = append(resultados, res)
	}

	// Renderizar template ou retornar JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resultados)
}

func processarAnamnese(w http.ResponseWriter, r *http.Request, pacienteID int) {
	// 1. Validar pacienteID
	if pacienteID <= 0 {
		log.Println("ID do paciente inválido:", pacienteID)
		http.Error(w, "ID do paciente inválido", http.StatusBadRequest)
		return
	}

	// 2. Parse do formulário
	if err := r.ParseForm(); err != nil {
		log.Println("Erro ao analisar formulário:", err)
		http.Error(w, "Erro ao processar formulário", http.StatusBadRequest)
		return
	}

	// 3. Validar campos obrigatórios
	dataColeta := r.FormValue("data_coleta")
	if dataColeta == "" {
		log.Println("Data de coleta não fornecida")
		http.Error(w, "Data de coleta é obrigatória", http.StatusBadRequest)
		return
	}

	// 4. Preparar dados
	anamnese := Anamnese{
		PacienteID:              pacienteID,
		MotivoExame:             r.Form["motivo"],
		FezPreventivo:           sql.NullString{String: r.FormValue("preventivo"), Valid: r.FormValue("preventivo") != ""},
		UltimoExameAno:          sql.NullString{String: r.FormValue("ultimo_exame_ano"), Valid: r.FormValue("ultimo_exame_ano") != ""},
		UsaDIU:                  sql.NullString{String: r.FormValue("diu"), Valid: r.FormValue("diu") != ""},
		Gravida:                 sql.NullString{String: r.FormValue("gravida"), Valid: r.FormValue("gravida") != ""},
		UsaPilula:               sql.NullString{String: r.FormValue("pilula"), Valid: r.FormValue("pilula") != ""},
		UsaHormonioMenopausa:    sql.NullString{String: r.FormValue("menopausa"), Valid: r.FormValue("menopausa") != ""},
		FezRadioterapia:         sql.NullString{String: r.FormValue("radio"), Valid: r.FormValue("radio") != ""},
		UltimaMenstruacao:       parseNullDate(r.FormValue("ultima_menstruacao")),
		NaoLembraMenstruacao:    r.FormValue("nao_lembra_menstruacao") == "on",
		SangramentoPosSexo:      sql.NullString{String: r.FormValue("sangramento_sexual"), Valid: r.FormValue("sangramento_sexual") != ""},
		SangramentoPosMenopausa: sql.NullString{String: r.FormValue("sangramento_menopausa"), Valid: r.FormValue("sangramento_menopausa") != ""},
		DataColeta:              parseDate(dataColeta),
	}

	// 5. Inserir no banco de dados
	query := `INSERT INTO anamneses (
        paciente_id, motivo, fez_preventivo,
        ultimo_exame_ano, usa_diu, gravida, usa_pilula, usa_hormonio_menopausa,
        fez_radioterapia, ultima_menstruacao, nao_lembra_menstruacao,
        sangramento_pos_sexo, sangramento_pos_menopausa, data_coleta
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := db.ExecContext(r.Context(), query,
		anamnese.PacienteID,
		pq.Array(anamnese.MotivoExame), // Converter array para formato PostgreSQL
		anamnese.FezPreventivo,
		anamnese.UltimoExameAno,
		anamnese.UsaDIU,
		anamnese.Gravida,
		anamnese.UsaPilula,
		anamnese.UsaHormonioMenopausa,
		anamnese.FezRadioterapia,
		anamnese.UltimaMenstruacao,
		anamnese.NaoLembraMenstruacao,
		anamnese.SangramentoPosSexo,
		anamnese.SangramentoPosMenopausa,
		anamnese.DataColeta,
	)

	if err != nil {
		log.Printf("Erro ao salvar anamnese: %v\nQuery: %s\n", err, query)
		http.Error(w, "Erro ao salvar anamnese no banco de dados", http.StatusInternalServerError)
		return
	}

	// 6. Redirecionar com mensagem de sucesso
	http.Redirect(w, r, "/static/Cadastrar exames/cadastroExame.html?success=Anamnese+salva+com+sucesso", http.StatusSeeOther)
}

// Funções auxiliares

// parseDate analisa uma string de data no formato YYYY-MM-DD
func parseDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}
	}
	return t
}

// parseNullDate analisa uma string de data para sql.NullTime
func parseNullDate(dateStr string) sql.NullTime {
	if dateStr == "" {
		return sql.NullTime{Valid: false}
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: t, Valid: true}
}

// Função auxiliar para parse de tempo
func parseTime(timeStr string) time.Time {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}
	}
	return t
}

func resultadoPacienteHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Obter CPF da query string
	cpf := r.URL.Query().Get("cpf")
	if cpf == "" {
		http.Error(w, "CPF não fornecido", http.StatusBadRequest)
		return
	}

	// 2. Remover formatação do CPF
	cpf = strings.ReplaceAll(cpf, ".", "")
	cpf = strings.ReplaceAll(cpf, "-", "")
	cpf = strings.ReplaceAll(cpf, " ", "")

	if len(cpf) != 11 {
		http.Error(w, "CPF deve conter 11 dígitos", http.StatusBadRequest)
		return
	}

	// 3. Buscar o paciente pelo CPF
	var pacienteID int
	err := db.QueryRow(`
        SELECT id FROM pacientes 
        WHERE REPLACE(REPLACE(cpf, '.', ''), '-', '') = $1`, cpf).Scan(&pacienteID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Paciente não encontrado", http.StatusNotFound)
		} else {
			http.Error(w, "Erro ao buscar paciente", http.StatusInternalServerError)
		}
		return
	}

	// 4. Buscar o diagnóstico mais recente com tratamento especial para arrays
	var (
		dentroLimitesNormais     sql.NullBool
		alteracoesBenignasStr    string
		outrasAlteracoesBenignas sql.NullString
		celulasAtipicasStr       string
		atipiasCelularesStr      string
		outrasNeoplasias         sql.NullString
		celulasEndometriais      sql.NullBool
		observacoesGerais        sql.NullString
		screeningCitotecnico     sql.NullString
		responsavel              string
		dataResultado            time.Time
	)

	err = db.QueryRow(`
        SELECT 
            d.dentro_limites_normais,
            d.alteracoes_benignas,
            d.outras_alteracoes_benignas,
            d.celulas_atipicas,
            d.atipias_celulares,
            d.outras_neoplasias,
            d.celulas_endometriais,
            d.observacoes_gerais,
            d.screening_citotecnico,
            d.responsavel,
            d.data_resultado
        FROM diagnosticos d
        JOIN exames_citopatologicos e ON d.exame_id = e.id
        WHERE e.paciente_id = $1
        ORDER BY d.data_resultado DESC
        LIMIT 1`, pacienteID).Scan(
		&dentroLimitesNormais,
		&alteracoesBenignasStr,
		&outrasAlteracoesBenignas,
		&celulasAtipicasStr,
		&atipiasCelularesStr,
		&outrasNeoplasias,
		&celulasEndometriais,
		&observacoesGerais,
		&screeningCitotecnico,
		&responsavel,
		&dataResultado,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Nenhum resultado encontrado para este paciente", http.StatusNotFound)
		} else {
			log.Printf("Erro ao buscar diagnóstico: %v", err)
			http.Error(w, "Erro ao buscar diagnóstico", http.StatusInternalServerError)
		}
		return
	}

	// Função para converter string de array para slice
	parseArray := func(s string) []string {
		if s == "" {
			return []string{}
		}
		// Remove possíveis {} e espaços
		s = strings.Trim(s, "{}")
		if s == "" {
			return []string{}
		}
		return strings.Split(s, ",")
	}

	// Criar struct de resposta
	diagnostico := struct {
		DentroLimitesNormais     sql.NullBool   `json:"dentro_limites_normais"`
		AlteracoesBenignas       []string       `json:"alteracoes_benignas"`
		OutrasAlteracoesBenignas sql.NullString `json:"outras_alteracoes_benignas"`
		CelulasAtipicas          []string       `json:"celulas_atipicas"`
		AtipiasCelulares         []string       `json:"atipias_celulares"`
		OutrasNeoplasias         sql.NullString `json:"outras_neoplasias"`
		CelulasEndometriais      sql.NullBool   `json:"celulas_endometriais"`
		ObservacoesGerais        sql.NullString `json:"observacoes_gerais"`
		ScreeningCitotecnico     sql.NullString `json:"screening_citotecnico"`
		Responsavel              string         `json:"responsavel"`
		DataResultado            time.Time      `json:"data_resultado"`
	}{
		DentroLimitesNormais:     dentroLimitesNormais,
		AlteracoesBenignas:       parseArray(alteracoesBenignasStr),
		OutrasAlteracoesBenignas: outrasAlteracoesBenignas,
		CelulasAtipicas:          parseArray(celulasAtipicasStr),
		AtipiasCelulares:         parseArray(atipiasCelularesStr),
		OutrasNeoplasias:         outrasNeoplasias,
		CelulasEndometriais:      celulasEndometriais,
		ObservacoesGerais:        observacoesGerais,
		ScreeningCitotecnico:     screeningCitotecnico,
		Responsavel:              responsavel,
		DataResultado:            dataResultado,
	}

	// Retornar como JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(diagnostico); err != nil {
		log.Printf("Erro ao codificar resposta: %v", err)
		http.Error(w, "Erro ao processar resposta", http.StatusInternalServerError)
	}
	
}
