package hana

// PathStat type
type PathStat struct {
	Directory    bool
	ReadOnly     bool
	Executable   bool
	Hidden       bool
	Archive      bool
	SymbolicLink bool
	Activated    bool
	TimeStamp    int64
}
