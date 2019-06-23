/*
 * Copyright (c) 2019. Pandranki Global Private Limited
 */

package bongo

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"testing"
	"time"
)

type Parent struct {
	DocumentBase `bson:",inline"`
	Bar          string
	Number       int
	FooBar       string
	Children     []ChildRef
	Child        ChildRef
	ChildProp    string `bson:"childProp"`
	diffTracker  *DiffTracker
}

func (f *Parent) GetDiffTracker() *DiffTracker {
	v := reflect.ValueOf(f.diffTracker)
	if !v.IsValid() || v.IsNil() {
		f.diffTracker = NewDiffTracker(f)
	}

	return f.diffTracker
}

type Child struct {
	DocumentBase `bson:",inline"`
	ParentID     primitive.ObjectID `bson:",omitempty"`
	Name         string
	SubChild     SubChildRef `bson:"subChild"`
	ChildProp    string
	diffTracker  *DiffTracker
}

func (c *Child) GetCascade(collection *Collection) []*CascadeConfig {

	ref := ChildRef{
		ID:       c.ID,
		Name:     c.Name,
		SubChild: c.SubChild,
	}
	connection := collection.Connection
	cascadeSingle := &CascadeConfig{
		Collection:  connection.Collection("parents"),
		Properties:  []string{"_ID", "name", "subChild.foo", "subChild._ID"},
		Data:        ref,
		ThroughProp: "child",
		RelType:     REL_ONE,
		Query: bson.M{
			"_ID": c.ParentID,
		},
	}

	cascadeCopy := &CascadeConfig{
		Collection: connection.Collection("parents"),
		Properties: []string{"childProp"},
		Data: map[string]interface{}{
			"childProp": c.ChildProp,
		},
		RelType: REL_ONE,
		Query: bson.M{
			"_ID": c.ParentID,
		},
	}

	cascadeMulti := &CascadeConfig{
		Collection:  connection.Collection("parents"),
		Properties:  []string{"_ID", "name", "subChild.foo", "subChild._ID"},
		Data:        ref,
		ThroughProp: "children",
		RelType:     REL_MANY,
		Query: bson.M{
			"_ID": c.ParentID,
		},
	}

	if c.GetDiffTracker().Modified("ParentID") {

		origID, _ := c.diffTracker.GetOriginalValue("ParentID")
		if origID != nil {
			oldQuery := bson.M{
				"_ID": origID,
			}
			cascadeSingle.OldQuery = oldQuery
			cascadeCopy.OldQuery = oldQuery
			cascadeMulti.OldQuery = oldQuery
		}

	}

	return []*CascadeConfig{cascadeSingle, cascadeMulti, cascadeCopy}
}

func (f *Child) GetDiffTracker() *DiffTracker {
	v := reflect.ValueOf(f.diffTracker)
	if !v.IsValid() || v.IsNil() {
		f.diffTracker = NewDiffTracker(f)
	}

	return f.diffTracker
}

type SubChild struct {
	DocumentBase `bson:",inline"`
	Foo          string
	ChildID      primitive.ObjectID
}

func (c *SubChild) GetCascade(collection *Collection) []*CascadeConfig {
	ref := SubChildRef{
		ID:  c.ID,
		Foo: c.Foo,
	}
	connection := collection.Connection
	cascadeSingle := &CascadeConfig{
		Collection:  connection.Collection("children"),
		Properties:  []string{"_ID", "foo"},
		Data:        ref,
		ThroughProp: "subChild",
		RelType:     REL_ONE,
		Query: bson.M{
			"_ID": c.ChildID,
		},
		Nest:     true,
		Instance: &Child{},
	}

	return []*CascadeConfig{cascadeSingle}
}

type SubChildRef struct {
	ID  primitive.ObjectID `bson:"_ID,omitempty"`
	Foo string
}

type ChildRef struct {
	ID       primitive.ObjectID `bson:"_ID,omitempty"`
	Name     string
	SubChild SubChildRef
}

func TestCascade(t *testing.T) {
	connection := getConnection()
	// defer connection.Session.Close()

	Convey("Cascade Save/Delete - full run through", t, func() {
		_ = connection.Session.Database("bongotest").Drop(context.Background())
		collection := connection.Collection("parents")

		childCollection := connection.Collection("children")
		subchildCollection := connection.Collection("subchildren")
		parent := &Parent{
			Bar:    "Testy McGee",
			Number: 5,
		}

		parent2 := &Parent{
			Bar:    "Other Parent",
			Number: 10,
		}

		err := collection.Save(parent)
		So(err, ShouldEqual, nil)
		err = collection.Save(parent2)
		So(err, ShouldEqual, nil)

		child := &Child{
			ParentID:  parent.ID,
			Name:      "Foo McGoo",
			ChildProp: "Doop McGoop",
		}
		err = childCollection.Save(child)

		// Wait a sec for the go routine to finish.
		time.Sleep(100 * time.Millisecond)

		So(err, ShouldEqual, nil)

		child.GetDiffTracker().Reset()
		newParent := &Parent{}
		_ = collection.FindByID(parent.ID, newParent)

		So(newParent.Child.Name, ShouldEqual, "Foo McGoo")
		So(newParent.Child.ID.Hex(), ShouldEqual, child.ID.Hex())
		So(newParent.Children[0].Name, ShouldEqual, "Foo McGoo")
		So(newParent.Children[0].ID.Hex(), ShouldEqual, child.ID.Hex())

		// No through prop should populate directly o the parent
		So(newParent.ChildProp, ShouldEqual, "Doop McGoop")

		// Now change the child parent ID...
		child.ParentID = parent2.ID
		So(child.GetDiffTracker().Modified("ParentID"), ShouldEqual, true)

		err = childCollection.Save(child)
		So(err, ShouldEqual, nil)

		// Wait a sec for the go routine to finish.
		time.Sleep(100 * time.Millisecond)

		child.diffTracker.Reset()
		// Now make sure it says the parent ID DIDNT change, because we just reset the tracker
		So(child.GetDiffTracker().Modified("ParentID"), ShouldEqual, false)

		newParent1 := &Parent{}
		collection.FindByID(parent.ID, newParent1)
		So(newParent1.Child.Name, ShouldEqual, "")
		So(newParent1.ChildProp, ShouldEqual, "")
		So(len(newParent1.Children), ShouldEqual, 0)
		newParent2 := &Parent{}
		collection.FindByID(parent2.ID, newParent2)
		So(newParent2.ChildProp, ShouldEqual, "Doop McGoop")
		So(newParent2.Child.Name, ShouldEqual, "Foo McGoo")
		So(newParent2.Child.ID.Hex(), ShouldEqual, child.ID.Hex())
		So(newParent2.Children[0].Name, ShouldEqual, "Foo McGoo")
		So(newParent2.Children[0].ID.Hex(), ShouldEqual, child.ID.Hex())

		// Make a new sub child, save it, and it should cascade to the child AND the parent
		subChild := &SubChild{
			Foo:     "MySubChild",
			ChildID: child.ID,
		}

		err = subchildCollection.Save(subChild)
		So(err, ShouldEqual, nil)

		// Wait a sec for the go routine to finish.
		time.Sleep(100 * time.Millisecond)

		// Fetch the parent
		newParent3 := &Parent{}
		collection.FindByID(parent2.ID, newParent3)
		So(newParent3.Child.SubChild.Foo, ShouldEqual, "MySubChild")
		So(newParent3.Child.SubChild.ID.Hex(), ShouldEqual, subChild.ID.Hex())

		newParent4 := &Parent{}
		_, err = childCollection.DeleteDocument(child)

		// Wait a sec for the go routine to finish.
		time.Sleep(100 * time.Millisecond)

		So(err, ShouldEqual, nil)
		collection.FindByID(parent2.ID, newParent4)
		So(newParent4.Child.Name, ShouldEqual, "")
		So(newParent4.ChildProp, ShouldEqual, "")
		So(len(newParent4.Children), ShouldEqual, 0)

	})

	Convey("MapFromCascadeProperties", t, func() {
		parent := &Parent{
			Bar: "bar",
			Child: ChildRef{
				Name: "child",
				SubChild: SubChildRef{
					Foo: "foo",
				},
			},
			Number: 5,
		}

		props := []string{"bar", "child.name"}

		mp := MapFromCascadeProperties(props, parent)

		So(len(mp), ShouldEqual, 2)
		So(mp["bar"], ShouldEqual, "bar")

		submp := mp["child"].(map[string]interface{})
		So(submp["name"], ShouldEqual, "child")

	})

}
