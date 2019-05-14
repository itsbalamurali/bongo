package bongo

import (
	. "github.com/smartystreets/goconvey/convey"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"context"
	"testing"
)

func TestValidation(t *testing.T) {
	Convey("Validation", t, func() {
		Convey("ValidateRequired()", func() {
			So(ValidateRequired("foo"), ShouldEqual, true)
			So(ValidateRequired(""), ShouldEqual, false)
			So(ValidateRequired(0), ShouldEqual, false)
			So(ValidateRequired(1), ShouldEqual, true)
		})

		Convey("ValidateInclusionIn()", func() {
			So(ValidateInclusionIn("foo", []string{"foo", "bar", "baz"}), ShouldEqual, true)
			So(ValidateInclusionIn("bing", []string{"foo", "bar", "baz"}), ShouldEqual, false)
		})

		Convey("ValidateMongoIdRef()", func() {
			connection := getConnection()

			defer func() {
				_ = connection.Session.Database("bongotest").Drop(context.Background())
			}()

			// Make the doc

			doc := &noHookDocument{}

			err := connection.Collection("docs").Save(doc)

			So(err, ShouldEqual, nil)
			So(ValidateMongoIdRef(doc.ID, connection.Collection("docs")), ShouldEqual, true)
			So(ValidateMongoIdRef(primitive.NewObjectID(), connection.Collection("docs")), ShouldEqual, false)
			So(ValidateMongoIdRef(primitive.NewObjectID(), connection.Collection("other_collection")), ShouldEqual, false)

		})
	})
}
