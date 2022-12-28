package swagger_ci

type Info struct {
}

type Path = map[string]*Body

type Body struct {
	Description string
	Consumes    []string
	Produces    []string
	Tags        []string
	Summary     string
	Parameters  []*Propertie
	Responses   map[string]*Response
}

type Propertie struct {
	Type        string
	Description string
	Name        string
	In          string
	Schema      *Schema
	Default     interface{}
	Enum        []interface{}
}

type Schema struct {
	Ref string `json:"$ref"`
}

type Response struct {
	Description string
	Schema      map[string]interface{}
}

type Definition struct {
	Type       string
	Required   []string
	Properties map[string]*Propertie
}

type Swagger struct {
	Swagger     string
	Info        *Info
	Paths       map[string]Path
	Definitions map[string]*Definition
}
