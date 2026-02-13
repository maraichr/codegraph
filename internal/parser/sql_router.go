package parser

// SQLRouter routes .sql files to the appropriate dialect parser based on FileInput.Language.
type SQLRouter struct {
	tsql  Parser
	pgsql Parser
}

func NewSQLRouter(tsql, pgsql Parser) *SQLRouter {
	return &SQLRouter{tsql: tsql, pgsql: pgsql}
}

func (r *SQLRouter) Parse(input FileInput) (*ParseResult, error) {
	if input.Language == "tsql" {
		return r.tsql.Parse(input)
	}
	return r.pgsql.Parse(input)
}

func (r *SQLRouter) Languages() []string {
	return []string{"tsql", "pgsql", "sql"}
}
