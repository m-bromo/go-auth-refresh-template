package config

type Config struct {
	Environment string `env:"ENVIRONMENT,default=development"`
	API         API
	Postgres    Postgres
	Jwt         Jwt
}

type API struct {
	Host string `env:"API_HOST,default=localhost"`
	Port string `env:"API_PORT,default=8080"`
}

type Postgres struct {
	Host     string `env:"POSTGRES_HOST,default=localhost"`
	Port     string `env:"POSTGRES_POST,default=5432"`
	Name     string `env:"POSTGRES_NAME,default=postgres_db"`
	User     string `env:"POSTGRES_USER,default=admin"`
	Password string `env:"POSTGRES_PASSWORD,default=password"`
}

type Jwt struct {
	PrivateKey string `env:"JWT_PRIVATE_KEY"`
	PublicKey  string `env:"JWT_PUBLIC_KEY"`
}
