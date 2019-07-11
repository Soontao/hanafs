package hana

type DirectoryMeta struct {
	Name             string                   `json:"Name"`
	ID               string                   `json:"Id"`
	Location         string                   `json:"Location"`
	ContentLocation  string                   `json:"ContentLocation"`
	ChildrenLocation string                   `json:"ChildrenLocation"`
	Directory        bool                     `json:"Directory"`
	Attributes       DirectoryAttributes      `json:"Attributes"`
	Workspaces       []interface{}            `json:"Workspaces"`
	SapBackPack      DirectoryMetaSapBackPack `json:"SapBackPack"`
	Parents          []interface{}            `json:"Parents"`
	LocalTimeStamp   int64                    `json:"LocalTimeStamp"`
}

type DirectoryAttributes struct {
	ReadOnly     bool                     `json:"ReadOnly"`
	Executable   bool                     `json:"Executable"`
	Hidden       bool                     `json:"Hidden"`
	Archive      bool                     `json:"Archive"`
	SymbolicLink bool                     `json:"SymbolicLink"`
	SapBackPack  DirAttributesSapBackPack `json:"SapBackPack"`
}

type DirAttributesSapBackPack struct {
	Structural bool `json:"Structural"`
}

type DirectoryMetaSapBackPack struct {
}

type DirectoryDetail struct {
	Name             string                    `json:"Name"`
	ID               string                    `json:"Id"`
	Location         string                    `json:"Location"`
	ContentLocation  string                    `json:"ContentLocation"`
	ChildrenLocation string                    `json:"ChildrenLocation"`
	ExportLocation   string                    `json:"ExportLocation"`
	ImportLocation   string                    `json:"ImportLocation"`
	Directory        bool                      `json:"Directory"`
	Attributes       DirectoryDetailAttributes `json:"Attributes"`
	Workspaces       []interface{}             `json:"Workspaces"`
	SapBackPack      ChildSapBackPack          `json:"SapBackPack"`
	Parents          []DirectoryDetailParent   `json:"Parents"`
	Children         []Child                   `json:"Children"`
}

type DirectoryDetailAttributes struct {
	ReadOnly     bool                                 `json:"ReadOnly"`
	Executable   bool                                 `json:"Executable"`
	Hidden       bool                                 `json:"Hidden"`
	Archive      bool                                 `json:"Archive"`
	SymbolicLink bool                                 `json:"SymbolicLink"`
	SapBackPack  DirectoryDetailAttributesSapBackPack `json:"SapBackPack"`
}

type DirectoryDetailAttributesSapBackPack struct {
	Structural bool `json:"Structural"`
}

type Child struct {
	Name             string        `json:"Name"`
	ID               string        `json:"Id"`
	Location         string        `json:"Location"`
	ContentLocation  string        `json:"ContentLocation"`
	ChildrenLocation string        `json:"ChildrenLocation"`
	ExportLocation   string        `json:"ExportLocation"`
	ImportLocation   string        `json:"ImportLocation"`
	Directory        bool          `json:"Directory"`
	Attributes       Attributes    `json:"Attributes"`
	Workspaces       []interface{} `json:"Workspaces"`
	// for file, sap back pack is different
	// SapBackPack      ChildSapBackPack `json:"SapBackPack"`
}

type ChildSapBackPack struct {
	SrcSystem        Responsible      `json:"SrcSystem"`
	DeliveryUnit     *string          `json:"DeliveryUnit,omitempty"`
	Vendor           *string          `json:"Vendor,omitempty"`
	Responsible      Responsible      `json:"Responsible"`
	OriginalLanguage OriginalLanguage `json:"OriginalLanguage"`
	Description      *string          `json:"Description,omitempty"`
}

type DirectoryDetailParent struct {
	Name             string `json:"Name"`
	ChildrenLocation string `json:"ChildrenLocation"`
	Location         string `json:"Location"`
	ExportLocation   string `json:"ExportLocation"`
}

type OriginalLanguage string

const (
	En   OriginalLanguage = "en"
	EnUS OriginalLanguage = "en_US"
)

type Responsible string

const (
	Sap Responsible = "SAP"
)
