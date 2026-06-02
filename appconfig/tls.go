package appconfig

// TLS holds certificate paths for one side of a TLS connection.
type TLS struct {
	CA   string `mapstructure:"ca"`
	Cert string `mapstructure:"cert"`
	Key  string `mapstructure:"key"`
}

// GRPCConn is the outbound gRPC connection TLS configuration.
// It is embedded under mapstructure key "conn" in ServiceConfig.
type GRPCConn struct {
	TLS         `mapstructure:",squash"`
	VerifyCerts bool `mapstructure:"verify_certs"`
	Client      TLS  `mapstructure:"client"`
}

// ValidateTLS returns an error if any required certificate path is missing.
// Call only when VerifyCerts is true.
func ValidateTLS(prefix string, t TLS) error {
	if t.CA == "" {
		return configError(prefix + ".ca is required when verify_certs is true")
	}
	if t.Cert == "" {
		return configError(prefix + ".cert is required when verify_certs is true")
	}
	if t.Key == "" {
		return configError(prefix + ".key is required when verify_certs is true")
	}
	return nil
}

// ValidateGRPCConn validates the GRPCConn TLS settings when verify_certs is enabled.
func ValidateGRPCConn(prefix string, conn GRPCConn) error {
	if conn.VerifyCerts {
		return ValidateTLS(prefix, conn.TLS)
	}
	return nil
}
