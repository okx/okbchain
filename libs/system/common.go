package system

const (
	Chain                = "brczero"
	TestnetPrefix        = "brczerotest"
	AppName              = "BRCZero"
	Server               = Chain + "d"
	Client               = Chain + "cli"
	ServerHome           = "$HOME/." + Server
	ClientHome           = "$HOME/." + Client
	ServerLog            = Server + ".log"
	EnvPrefix            = "BRCZero"
	CoinType      uint32 = 60
	Currency             = "brczero"
)
