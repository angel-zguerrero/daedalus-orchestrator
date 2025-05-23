package config

type Config struct {
	TTLInternalError    uint64
	DefaultRootUser     string
	DefaultRootPassword string
}

type ConfigFromMap struct {
	port                  int
	db_name               string
	default_root_user     string
	default_root_password string
}

func ConfigFromMapToConfig(configFromMap ConfigFromMap) *Config {
	c := &Config{
		DefaultRootUser:     configFromMap.default_root_user,
		DefaultRootPassword: configFromMap.default_root_password,
	}
	return c
}

var GlobalConfiguration *Config
