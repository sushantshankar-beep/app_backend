package config

import "os"

type Config struct {
	// Existing
	MongoURI  string
	DBName    string
	JWTSecret string
	HTTPPort  string

	// 🔹 Payment (PayU)
	PayUKey     string
	PayUSalt    string
	PayUBaseURL string
	BaseURL     string
}

func Load() Config {
	return Config{
		MongoURI:  getEnv("MONGO_URI", "mongodb+srv://jaswantkushwahadev_db_user:jassi2002@cluster0.x8bcmgh.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0"),
		DBName:    getEnv("DB_NAME", "test"),
		JWTSecret: getEnv("JWT_SECRET", "fde45re4567yuio098u7ytfrdsaq12345t6yu"),
		HTTPPort:  getEnv("PORT", "8080"),
		PayUKey:     getEnv("PAYU_KEY", ""),
		PayUSalt:    getEnv("PAYU_SALT", ""),
		PayUBaseURL: getEnv("PAYU_BASE_URL", "https://secure.payu.in"),
		BaseURL:     getEnv("BASE_URL", "http://localhost:8080"),
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
