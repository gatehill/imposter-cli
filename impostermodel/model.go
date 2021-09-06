package impostermodel

type ResponseConfig struct {
	ResponseCode string `json:"responseCode,omitempty"`
	StaticFile   string `json:"staticFile,omitempty"`
	StaticData   string `json:"staticData,omitempty"`
	ExampleName  string `json:"exampleName,omitempty"`
	ScriptFile   string `json:"scriptFile,omitempty"`
}

type Resource struct {
	Path     string          `json:"path"`
	Method   string          `json:"method"`
	Response *ResponseConfig `json:"response,omitempty"`
}

type PluginConfig struct {
	Plugin    string          `json:"plugin"`
	SpecFile  string          `json:"specFile"`
	Response  *ResponseConfig `json:"response,omitempty"`
	Resources []Resource      `json:"resources,omitempty"`
}
