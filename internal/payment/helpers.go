package payment

import (
	"regexp"
	"strconv"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func parseAmount(v string) (float64, error) {
	clean := regexp.MustCompile(`[^\d.]`).ReplaceAllString(v, "")
	return strconv.ParseFloat(clean, 64)
}

func toObjectID(v any) primitive.ObjectID {
	id, _ := primitive.ObjectIDFromHex(v.(string))
	return id
}
