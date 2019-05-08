package bongo

import (
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"context"
	"testing"
)

type noHookDocument struct {
	DocumentBase `bson:",inline"`
	Name         string
}

type hookedDocument struct {
	DocumentBase    `bson:",inline"`
	RanBeforeSave   bool
	RanAfterSave    bool
	RanBeforeDelete bool
	RanAfterDelete  bool
	RanAfterFind    bool
}

func (h *hookedDocument) BeforeSave(c *Collection) error {
	h.RanBeforeSave = true
	So(c.Context.Get("foo"), ShouldEqual, "bar")
	return nil
}

func (h *hookedDocument) AfterSave(c *Collection) error {
	h.RanAfterSave = true
	So(c.Context.Get("foo"), ShouldEqual, "bar")
	return nil
}

func (h *hookedDocument) BeforeDelete(c *Collection) error {
	h.RanBeforeDelete = true
	So(c.Context.Get("foo"), ShouldEqual, "bar")
	return nil
}

func (h *hookedDocument) AfterDelete(c *Collection) error {
	h.RanAfterDelete = true
	So(c.Context.Get("foo"), ShouldEqual, "bar")
	return nil
}

func (h *hookedDocument) AfterFind(c *Collection) error {
	h.RanAfterFind = true
	So(c.Context.Get("foo"), ShouldEqual, "bar")
	return nil
}

type validatedDocument struct {
	DocumentBase `bson:",inline"`
	Name         string
}

func (v *validatedDocument) Validate(c *Collection) []error {
	return []error{errors.New("test validation error")}
}

func TestCollection(t *testing.T) {

	conn := getConnection()

	Convey("Saving", t, func() {
		Convey("should be able to save a document with no hooks, update id, and use new tracker", func() {

			doc := &noHookDocument{}
			doc.Name = "foo"
			So(doc.IsNew(), ShouldEqual, true)

			err := conn.Collection("tests").Save(doc)
			So(err, ShouldEqual, nil)
			So(doc.ID.IsZero(), ShouldEqual, true)
			So(doc.IsNew(), ShouldEqual, false)
		})

		Convey("should be able to save a document with save hooks", func() {
			doc := &hookedDocument{}

			err := conn.Collection("tests").Save(doc)

			So(err, ShouldEqual, nil)
			So(doc.RanBeforeSave, ShouldEqual, true)
			So(doc.RanAfterSave, ShouldEqual, true)
		})

		Convey("should return a validation error if the validate method has things in the return value", func() {
			doc := &validatedDocument{}
			err := conn.Collection("tests").Save(doc)

			v, ok := err.(*ValidationError)
			So(ok, ShouldEqual, true)
			So(v.Errors[0].Error(), ShouldEqual, "test validation error")
		})

		Convey("should be able to save an existing document", func() {
			doc := &noHookDocument{}
			doc.Name = "foo"
			So(doc.IsNew(), ShouldEqual, true)

			err := conn.Collection("tests").Save(doc)
			So(err, ShouldEqual, nil)
			So(doc.ID.IsZero(), ShouldEqual, true)
			So(doc.IsNew(), ShouldEqual, false)

			err = conn.Collection("tests").Save(doc)

			So(err, ShouldEqual, nil)
			count, err := conn.Collection("tests").Collection().Count()
			So(err, ShouldEqual, nil)
			So(count, ShouldEqual, 1)
		})

		Convey("should set created and modified dates", func() {

			doc := &noHookDocument{}
			doc.Name = "foo"

			err := conn.Collection("tests").Save(doc)
			So(err, ShouldEqual, nil)
			So(doc.CreatedAt.UnixNano(), ShouldEqual, doc.GetUpdatedAt().UnixNano())

			err = conn.Collection("tests").Save(doc)
			So(err, ShouldEqual, nil)
			So(doc.UpdatedAt.UnixNano(), ShouldBeGreaterThan, doc.GetCreatedAt().UnixNano())
		})

		Reset(func() {
			conn.Session.Database("bongotest").Drop(context.Background())
		})
	})

	Convey("FindById", t, func() {
		doc := &noHookDocument{}
		err := conn.Collection("tests").Save(doc)
		So(err, ShouldEqual, nil)

		Convey("should find a doc by id", func() {
			newDoc := &noHookDocument{}
			err := conn.Collection("tests").FindById(doc.GetID(), newDoc)
			So(err, ShouldEqual, nil)
			So(newDoc.ID.Hex(), ShouldEqual, doc.ID.Hex())
		})

		Convey("should find a doc by id and run afterFind", func() {
			newDoc := &hookedDocument{}
			err := conn.Collection("tests").FindById(doc.GetID(), newDoc)
			So(err, ShouldEqual, nil)
			So(newDoc.ID.Hex(), ShouldEqual, doc.ID.Hex())
			So(newDoc.RanAfterFind, ShouldEqual, true)
		})

		Convey("should return a document not found error if doc not found", func() {

			err := conn.Collection("tests").FindById(primitive.NewObjectID(), doc)
			_, ok := err.(*DocumentNotFoundError)
			So(ok, ShouldEqual, true)
		})

		Reset(func() {
			conn.Session.Database("bongotest").Drop(context.Background())
		})
	})

	Convey("FindOne", t, func() {
		doc := &noHookDocument{}
		doc.Name = "foo"
		err := conn.Collection("tests").Save(doc)
		So(err, ShouldEqual, nil)

		Convey("should find one with query", func() {
			newDoc := &noHookDocument{}
			err := conn.Collection("tests").FindOne(bson.M{
				"name": "foo",
			}, newDoc)
			So(err, ShouldEqual, nil)
			So(newDoc.ID.Hex(), ShouldEqual, doc.ID.Hex())
		})

		Convey("should find one with query and run afterFind", func() {
			newDoc := &hookedDocument{}
			err := conn.Collection("tests").FindOne(bson.M{
				"name": "foo",
			}, newDoc)
			So(err, ShouldEqual, nil)
			So(newDoc.ID.Hex(), ShouldEqual, doc.ID.Hex())
			So(newDoc.RanAfterFind, ShouldEqual, true)
		})

		Reset(func() {
			conn.Session.Database("bongotest").Drop(context.Background())
		})
	})

	Convey("Delete", t, func() {
		Convey("should be able delete a document", func() {
			doc := &noHookDocument{}

			err := conn.Collection("tests").Save(doc)
			So(err, ShouldEqual, nil)

			_, err = conn.Collection("tests").DeleteDocument(doc)
			So(err, ShouldEqual, nil)

			count, err := conn.Collection("tests").Collection().Count()

			So(err, ShouldEqual, nil)
			So(count, ShouldEqual, 0)
		})

		Convey("should be able delete a document and run hooks", func() {
			doc := &hookedDocument{}

			err := conn.Collection("tests").Save(doc)
			So(err, ShouldEqual, nil)

			_, err = conn.Collection("tests").DeleteDocument(doc)
			So(err, ShouldEqual, nil)

			count, err := conn.Collection("tests").Collection().Count()

			So(err, ShouldEqual, nil)
			So(count, ShouldEqual, 0)

			So(doc.RanBeforeDelete, ShouldEqual, true)
			So(doc.RanAfterDelete, ShouldEqual, true)
		})

		Convey("should be able delete a document with DeleteOne", func() {
			doc := &noHookDocument{}

			err := conn.Collection("tests").Save(doc)
			So(err, ShouldEqual, nil)

			_,err = conn.Collection("tests").DeleteOne(bson.D{
				"_id": doc.ID,
			})
			So(err, ShouldEqual, nil)

			count, err := conn.Collection("tests").Collection().Count()

			So(err, ShouldEqual, nil)
			So(count, ShouldEqual, 0)
		})

		Convey("should be able delete a document with Delete", func() {
			doc := &noHookDocument{}

			err := conn.Collection("tests").Save(doc)
			So(err, ShouldEqual, nil)

			info, err := conn.Collection("tests").Delete(bson.D{
				"_id": doc.ID,
			})
			So(err, ShouldEqual, nil)
			So(info.DeletedCount, ShouldEqual, 1)

			count, err := conn.Collection("tests").Collection().Count()

			So(err, ShouldEqual, nil)
			So(count, ShouldEqual, 0)
		})

	})
}
