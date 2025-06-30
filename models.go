package main

import (
	"database/sql"
	"time"
)

type Usuario struct {
	ID        int            `db:"id"`
	Email     string         `db:"email"`
	Senha     string         `db:"senha"`
	Tipo      string         `db:"tipo"` // "profissional" ou "paciente"
	CNES      sql.NullString `db:"cnes"`
	CPF       sql.NullString `db:"cpf"`
	Telefone  sql.NullString `db:"telefone"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

type Paciente struct {
	// Identificação
	ID int `db:"id"`

	// Dados da Unidade de Saúde
	UF               string         `db:"uf"`
	CNES             string         `db:"cnes"`
	Protocolo        sql.NullString `db:"protocolo"`
	NomeUnidade      string         `db:"nome_unidade"`
	MunicipioUnidade string         `db:"municipio_unidade"`

	// Informações Pessoais
	CartaoSUS      string         `db:"cartao_sus"`
	NomeCompleto   string         `db:"nome_completo"`
	NomeMae        string         `db:"nome_mae"`
	CPF            sql.NullString `db:"cpf"`
	Apelido        sql.NullString `db:"apelido"`
	DataNascimento time.Time      `db:"data_nascimento"`
	Idade          sql.NullInt32  `db:"idade"`
	Etnia          sql.NullString `db:"etnia"`
	EtniaOutra     sql.NullString `db:"etnia_outra"`

	// Endereço
	Logradouro      sql.NullString `db:"logradouro"`
	Numero          sql.NullString `db:"numero"`
	Complemento     sql.NullString `db:"complemento"`
	Bairro          sql.NullString `db:"bairro"`
	UFEndereco      sql.NullString `db:"uf_endereco"`
	CodigoMunicipio sql.NullString `db:"codigo_municipio"`
	Municipio       sql.NullString `db:"municipio_endereco"`
	CEP             sql.NullString `db:"cep"`
	DDD             sql.NullString `db:"ddd"`
	Telefone        sql.NullString `db:"telefone"`
	PontoReferencia sql.NullString `db:"ponto_referencia"`
	Escolaridade    sql.NullString `db:"escolaridade"`

	// Controle
	CreatedAt time.Time `db:"created_at"`
}

type UnidadeSaude struct {
	ID          int
	UF          string
	CNES        string
	NomeUnidade string
	Municipio   string
	Protocolo   sql.NullString
	CreatedAt   time.Time
}

type Anamnese struct {
	ID                      int
	PacienteID              int
	MotivoExame             []string
	FezPreventivo           sql.NullString
	UltimoExameAno          sql.NullString
	UsaDIU                  sql.NullString
	Gravida                 sql.NullString
	UsaPilula               sql.NullString
	UsaHormonioMenopausa    sql.NullString
	FezRadioterapia         sql.NullString
	UltimaMenstruacao       sql.NullTime
	NaoLembraMenstruacao    bool
	SangramentoPosSexo      sql.NullString
	SangramentoPosMenopausa sql.NullString
	DataColeta              time.Time
	CreatedAt               time.Time
}

type ExameClinico struct {
	ID             int
	PacienteID     int
	ProfissionalID int
	InspecaoColo   sql.NullString
	SinaisDST      sql.NullBool
	Observacoes    sql.NullString
	DataColeta     time.Time
	Responsavel    string
	CreatedAt      time.Time
}

type Laboratorio struct {
	ID              int
	CNES            string
	NomeLaboratorio string
	CreatedAt       time.Time
}

type ExameCitopatologico struct {
	ID                     int
	PacienteID             int // Mantemos apenas o vínculo com o paciente
	LaboratorioID          int
	NumeroExame            string
	RecebidoEm             time.Time
	AmostraRejeitada       sql.NullString
	CausaRejeicao          sql.NullString
	Epitelios              []string
	AdequabilidadeMaterial sql.NullString
	MotivoInsatisfatorio   sql.NullString
	CreatedAt              time.Time
}

type Diagnostico struct {
	ID                       int
	ExameID                  int
	DentroLimitesNormais     sql.NullBool
	AlteracoesBenignas       []string
	OutrasAlteracoesBenignas sql.NullString
	CelulasAtipicas          sql.NullString
	AtipiasCelulares         []string
	OutrasNeoplasias         sql.NullString
	CelulasEndometriais      sql.NullBool
	ObservacoesGerais        sql.NullString
	ScreeningCitotecnico     sql.NullString
	Responsavel              string
	DataResultado            time.Time
	CreatedAt                time.Time
}

type Consulta struct {
	ID             int
	PacienteID     int
	ProfissionalID int
	Especialidade  string
	DataConsulta   time.Time
	Horario        time.Time
	LocalConsulta  string
	Status         string
	CreatedAt      time.Time
}
