package hana

type File struct {
	Name           string          `json:"Name"`
	Location       string          `json:"Location"`
	RunLocation    string          `json:"RunLocation"`
	Directory      bool            `json:"Directory"`
	LocalTimeStamp int64           `json:"LocalTimeStamp"`
	ContentType    string          `json:"ContentType"`
	Attributes     Attributes      `json:"Attributes"`
	ETag           string          `json:"ETag"`
	Parents        []Parent        `json:"Parents"`
	SapBackPack    FileSapBackPack `json:"SapBackPack"`
}

type Attributes struct {
	SapBackPack  AttributesSapBackPack `json:"SapBackPack"`
	ReadOnly     bool                  `json:"ReadOnly"`
	Executable   bool                  `json:"Executable"`
	Hidden       bool                  `json:"Hidden"`
	Archive      bool                  `json:"Archive"`
	SymbolicLink bool                  `json:"SymbolicLink"`
}

type AttributesSapBackPack struct {
	Activated  bool `json:"Activated"`
	IsDeletion bool `json:"IsDeletion"`
}

type Parent struct {
	Name             string `json:"Name"`
	ChildrenLocation string `json:"ChildrenLocation"`
	Location         string `json:"Location"`
	ExportLocation   string `json:"ExportLocation"`
}

type FileSapBackPack struct {
	Version      int64  `json:"Version"`
	Type         int64  `json:"Type"`
	ActivatedAt  int64  `json:"ActivatedAt"`
	ActivatedBy  string `json:"ActivatedBy"`
	ObjectStatus string `json:"ObjectStatus"`
	IsDeletion   bool   `json:"IsDeletion"`
}
