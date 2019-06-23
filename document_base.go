/*
 * Copyright (c) 2019. Pandranki Global Private Limited
 */

package bongo

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type DocumentBase struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	DeletedAt time.Time          `json:"deleted_at,omitempty" bson:"deleted_at,omitempty"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
	// We want this to default to false without any work. So this will be the opposite of isNew. We want it to be new unless set to existing
	exists bool
}

// Satisfy the new tracker interface
func (d *DocumentBase) SetIsNew(isNew bool) {
	d.exists = !isNew
}

// Is the document new
func (d *DocumentBase) IsNew() bool {
	return !d.exists
}

// Satisfy the document interface
func (d *DocumentBase) GetID() primitive.ObjectID {
	return d.ID
}

// Sets the ID for the document
func (d *DocumentBase) SetID(id primitive.ObjectID) {
	d.ID = id
}

// Set's the created date
func (d *DocumentBase) SetCreatedAt(t time.Time) {
	d.CreatedAt = t
}

// Get the created date
func (d *DocumentBase) GetCreatedAt() time.Time {
	return d.CreatedAt
}

// Sets the updated date
func (d *DocumentBase) SetUpdatedAt(t time.Time) {
	d.UpdatedAt = t
}

// Get's the updated date
func (d *DocumentBase) GetUpdatedAt() time.Time {
	return d.UpdatedAt
}

// Sets the deleted date
func (d *DocumentBase) SetDeletedAt(t time.Time) {
	d.DeletedAt = t
}

// Get's the deleted date
func (d *DocumentBase) GetDeletedAt() time.Time {
	return d.DeletedAt
}
