package config

import "fmt"

type DatabaseConfig struct {
	Host     string `env:"DB_HOST,required,notEmpty"`
	Port     int    `env:"DB_PORT,required,notEmpty"`
	Username string `env:"DB_USERNAME,required,notEmpty" json:"-" yaml:"-"`
	Password string `env:"DB_PASSWORD,required,notEmpty" json:"-" yaml:"-"`
	Name     string `env:"DB_NAME,required,notEmpty"`
}

func (d *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?target_session_attrs=read-write&sslmode=disable",
		d.Username,
		d.Password,
		d.Host,
		d.Port,
		d.Name,
	)
}
