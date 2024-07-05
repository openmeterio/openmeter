package redis

// Config stores the user provided configuration parameters
type Config struct {
	Address  string
	Database int
	Username string
	Password string
	Sentinel struct {
		Enabled    bool
		MasterName string
	}
	TLS struct {
		Enabled            bool
		InsecureSkipVerify bool
	}
}
