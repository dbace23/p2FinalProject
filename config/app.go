package config

type App struct {
	Port         string `env:"APP_PORT" default:"8080"`
	DatabaseURL  string `env:"DATABASE_URL,required"`
	JWTSecret    string `env:"JWT_SECRET,required"`
	ApiNinjasKey string `env:"API_NINJAS_KEY"`
	Env          string `env:"APP_ENV" default:"dev"`
}
