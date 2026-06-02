package appconfig

import "github.com/spf13/pflag"

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

// RegisterGRPCConnFlags registers gRPC outbound-connection TLS flags under the
// given prefix (e.g. "service.conn" or "service.grpc.conn").
// verifyDefault sets the default value for the verify_certs flag.
//
// Registered flags and their env equivalents (prefix = "service.conn"):
//
//	service.conn.verify_certs → SERVICE_CONN_VERIFY_CERTS
//	service.conn.ca           → SERVICE_CONN_CA
//	service.conn.cert         → SERVICE_CONN_CERT
//	service.conn.key          → SERVICE_CONN_KEY
//	service.conn.client.ca    → SERVICE_CONN_CLIENT_CA
//	service.conn.client.cert  → SERVICE_CONN_CLIENT_CERT
//	service.conn.client.key   → SERVICE_CONN_CLIENT_KEY
func RegisterGRPCConnFlags(fs *pflag.FlagSet, prefix string, verifyDefault bool) {
	fs.Bool(prefix+".verify_certs", verifyDefault, "Verify TLS certificates on outbound gRPC connections")
	fs.String(prefix+".ca", "", "CA certificate path")
	fs.String(prefix+".cert", "", "Server certificate path")
	fs.String(prefix+".key", "", "Server certificate key path")
	fs.String(prefix+".client.ca", "", "Client CA certificate path")
	fs.String(prefix+".client.cert", "", "Client certificate path")
	fs.String(prefix+".client.key", "", "Client certificate key path")
}
