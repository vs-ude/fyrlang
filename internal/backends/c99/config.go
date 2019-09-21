package c99

type (
	// Config contains the complete configuration of the c99 backend.
	Config struct {
		Compiler       *CompilerConf
		Archiver       *ArchiverConf
		PlatformHosted bool
		IntSizeBit     int
	}

	// CompilerConf contains the configuration of the c99 compiler.
	CompilerConf struct {
		Bin          string
		DebugFlags   string
		ReleaseFlags string
	}

	// ArchiverConf contains the configuration of the archiver.
	ArchiverConf struct {
		Bin   string
		Flags string
	}
)

// Name prints the name of the backend.
func (c *Config) Name() string {
	return "c99"
}

// Default configures the backend with the default values.
func (c *Config) Default() {
	c.Compiler = &CompilerConf{Bin: "gcc", DebugFlags: "-g", ReleaseFlags: "-O3"}
	c.Archiver = &ArchiverConf{Bin: "ar", Flags: "rcs"}
	c.PlatformHosted = true
	c.IntSizeBit = 32
}
