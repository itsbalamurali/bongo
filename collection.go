package bongo

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
	"strings"
)

type BeforeSaveHook interface {
	BeforeSave(*Collection) error
}

type AfterSaveHook interface {
	AfterSave(*Collection) error
}

type BeforeDeleteHook interface {
	BeforeDelete(*Collection) error
}

type AfterDeleteHook interface {
	AfterDelete(*Collection) error
}

type AfterFindHook interface {
	AfterFind(*Collection) error
}

type ValidateHook interface {
	Validate(*Collection) []error
}

type ValidationError struct {
	Errors []error
}

type TimeCreatedTracker interface {
	GetCreatedAt() time.Time
	SetCreatedAt(time.Time)
}

type TimeModifiedTracker interface {
	GetUpdatedAt() time.Time
	SetUpdatedAt(time.Time)
}

type Document interface {
	GetID() primitive.ObjectID
	SetID(primitive.ObjectID)
}

type CascadingDocument interface {
	GetCascade(*Collection) []*CascadeConfig
}

func (v *ValidationError) Error() string {
	errs := make([]string, len(v.Errors))

	for i, e := range v.Errors {
		errs[i] = e.Error()
	}
	return "Validation failed. (" + strings.Join(errs, ", ") + ")"
}

type Collection struct {
	Name       string
	Database   string
	Context    *Context
	Connection *Connection
}

type NewTracker interface {
	SetIsNew(bool)
	IsNew() bool
}

type DocumentNotFoundError struct{}

func (d DocumentNotFoundError) Error() string {
	return "Document not found"
}

// Collection ...
func (c *Collection) Collection() *mongo.Collection {
	return c.Connection.Session.Database(c.Database).Collection(c.Name)
}

// CollectionOnSession ...
func (c *Collection) collectionOnSession(sess *mongo.Client) *mongo.Collection {
	return sess.Database(c.Database).Collection(c.Name)
}

func (c *Collection) PreSave(doc Document) error {
	// Validate?
	if validator, ok := doc.(ValidateHook); ok {
		errs := validator.Validate(c)

		if len(errs) > 0 {
			return &ValidationError{errs}
		}
	}

	if hook, ok := doc.(BeforeSaveHook); ok {
		err := hook.BeforeSave(c)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Collection) Save(doc Document) error {
	var err error

	err = c.PreSave(doc)
	if err != nil {
		return err
	}
	// If the model implements the NewTracker interface, we'll use that to determine newness. Otherwise always assume it's new

	isNew := true
	if newt, ok := doc.(NewTracker); ok {
		isNew = newt.IsNew()
	}

	// Add created/modified time. Also set on the model itself if it has those fields.
	now := time.Now()

	if tt, ok := doc.(TimeCreatedTracker); ok && isNew {
		tt.SetCreatedAt(now)
	}

	if tt, ok := doc.(TimeModifiedTracker); ok {
		tt.SetUpdatedAt(now)
	}

	go CascadeSave(c, doc)

	id := doc.GetID()

	if !isNew && id.IsZero() {
		return errors.New("New tracker says this document isn't new but there is no valid Id field")
	}

	if isNew && id.IsZero() {
		// Generate an Id
		id = primitive.NewObjectID()
		doc.SetID(id)
	}

	err = c.UpsertID(id, doc)
	if err != nil {
		return err
	}

	if hook, ok := doc.(AfterSaveHook); ok {
		err = hook.AfterSave(c)
		if err != nil {
			return err
		}
	}

	// We saved it, no longer new
	if newt, ok := doc.(NewTracker); ok {
		newt.SetIsNew(false)
	}

	return nil
}

func (c *Collection) FindByID(id primitive.ObjectID, doc interface{}) error {

	filter := bson.D{{"_id", id}}

	err := c.Collection().FindOne(context.Background(), filter).Decode(&doc)

	// Handle errors coming from mgo - we want to convert it to a DocumentNotFoundError so people can figure out
	// what the error type is without looking at the text
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &DocumentNotFoundError{}
		} else {
			return err
		}
	}

	if hook, ok := doc.(AfterFindHook); ok {
		err = hook.AfterFind(c)
		if err != nil {
			return err
		}
	}

	// We retrieved it, so set new to false
	if newt, ok := doc.(NewTracker); ok {
		newt.SetIsNew(false)
	}
	return nil
}

// This doesn't actually do any DB interaction, it just creates the result set so we can
// start looping through on the iterator
func (c *Collection) Find(query interface{}) (*ResultSet,error) {
	col := c.Collection()

	// Count for testing
	cursor, err := col.Find(context.Background(),query)
	resultset := new(ResultSet)

	opts := &options.FindOptions{}

	resultset.Query = opts
	resultset.Cursor = cursor
	resultset.Params = query
	resultset.Collection = c

	return resultset, err
}

func (c *Collection) UpsertID(id primitive.ObjectID, doc interface{}) error {
	_, err := c.Collection().UpdateOne(context.Background(),bson.D{{"_id", id}}, doc)
	if err != nil {
		return err
	}
	return nil
}

func (c *Collection) FindOne(query interface{}, doc interface{}) error {

	// Now run a find
	results ,err  := c.Find(query)
	if err != nil{
		return err
	}
	results.Query.SetLimit(1)
	hasNext := results.Next(doc)
	if !hasNext {
		// There could have been an error fetching the next one, which would set the Error property on the resultset
		if results.Error != nil {
			return results.Error
		} else {
			return &DocumentNotFoundError{}
		}
	}

	if newt, ok := doc.(NewTracker); ok {
		newt.SetIsNew(false)
	}

	return nil
}

func (c *Collection) DeleteDocument(doc Document) (*mongo.DeleteResult, error) {
	var err error
	// Create a new session per mgo's suggestion to avoid blocking
	col := c.Collection()

	if hook, ok := doc.(BeforeDeleteHook); ok {
		err := hook.BeforeDelete(c)
		if err != nil {
			return nil, err
		}
	}

	res ,err := col.DeleteOne(context.Background(),bson.M{"_id": doc.GetID()})

	if err != nil {
		return nil, err
	}

	go CascadeDelete(c, doc)

	if hook, ok := doc.(AfterDeleteHook); ok {
		err = hook.AfterDelete(c)
		if err != nil {
			return nil, err
		}
	}

	return res, nil

}

// Convenience method which just delegates to mgo. Note that hooks are NOT run
func (c *Collection) Delete(query bson.D) (*mongo.DeleteResult, error) {
	col := c.Collection()
	return col.DeleteMany(context.Background(),query)
}

// Convenience method which just delegates to mgo. Note that hooks are NOT run
func (c *Collection) DeleteOne(query bson.D) (*mongo.DeleteResult, error) {
	col := c.Collection()
	return col.DeleteOne(context.Background(),query)
}
