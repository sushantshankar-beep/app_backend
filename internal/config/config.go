package config

import "os"

type Config struct {
	MongoURI  string
	DBName    string
	JWTSecret string
	HTTPPort  string
}

func Load() Config {
	return Config{
		MongoURI:  getEnv("MONGO_URI", ""),
		DBName:    getEnv("DB_NAME", ""),
		JWTSecret: getEnv("JWT_SECRET", ""),
		HTTPPort:  getEnv("PORT", "8080"),
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
