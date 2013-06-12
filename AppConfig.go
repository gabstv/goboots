package goboots

type AppConfig struct {
	Name            string
	Version         string
	HostAddr        string
	HostAddrTLS     string
	UseEnvPort      bool
	EnvPortIsTLS    bool
	MongoDbs        string
	Database        string
	Salt            string
	LocalePath      string
	DefaultLanguage string
	Data            map[string]string

	// TLS
	TLSCertificatePath string
	TLSKeyPath         string
	TLSRedirect        bool
}
