package bongo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
)

func ValidateRequired(val interface{}) bool {
	valueOf := reflect.ValueOf(val)
	return valueOf.Interface() != reflect.Zero(valueOf.Type()).Interface()
}

func ValidateMongoIdRef(id primitive.ObjectID, collection *Collection) bool {
	cur, err := collection.Collection().Find(context.Background(), bson.M{"_id": id})
	if err != nil || cur.Err() != nil || cur != nil {
		return false
	}

	return true
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func ValidateInclusionIn(value string, options []string) bool {
	return stringInSlice(value, options)
}
