package bongo

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math"
)

type ResultSet struct {
	Query      *options.FindOptions
	Cursor     *mongo.Cursor
	loadedIter bool
	Collection *Collection
	Error      error
	Params     interface{}
}

type PaginationInfo struct {
	Current       int `json:"current"`
	TotalPages    int `json:"totalPages"`
	PerPage       int `json:"perPage"`
	TotalRecords  int64 `json:"totalRecords"`
	RecordsOnPage int `json:"recordsOnPage"`
}

func (r *ResultSet) Next(doc interface{}) bool {

	// Check if the iter has been instantiated yet
	if !r.loadedIter {
		r.loadedIter = true
	}

	gotResult := r.Cursor.Next(context.Background())

	if gotResult {

		if hook, ok := doc.(AfterFindHook); ok {
			err := hook.AfterFind(r.Collection)
			if err != nil {
				r.Error = err
				return false
			}
		}

		if newt, ok := doc.(NewTracker); ok {
			newt.SetIsNew(false)
		}
		return true
	}

	err := r.Cursor.Err()
	if err != nil {
		r.Error = err
	}

	return false
}

func (r *ResultSet) Free() error {
	if r.loadedIter {
		if err := r.Cursor.Close(context.Background()); err != nil {
			return err
		}
	}

	return nil
}

// Set skip + limit on the current query and generates a PaginationInfo struct with info for your front end
func (r *ResultSet) Paginate(perPage, page int) (*PaginationInfo, error) {
	info := new(PaginationInfo)
	// Get count on a different session to avoid blocking
	sess := r.Collection.Connection.Session
	count, err := sess.Database(r.Collection.Database).Collection(r.Collection.Name).CountDocuments(context.Background(),nil)

	if err != nil {
		return info, err
	}

	// Calculate how many pages
	totalPages := int(math.Ceil(float64(count) / float64(perPage)))

	if page < 1 {
		page = 1
	} else if page > totalPages {
		page = totalPages
	}

	skip := (page - 1) * perPage

	r.Query.SetSkip(int64(skip)).SetLimit(int64(perPage))

	info.TotalPages = totalPages
	info.PerPage = perPage
	info.Current = page
	info.TotalRecords = count

	if info.Current < info.TotalPages {
		info.RecordsOnPage = info.PerPage
	} else {

		info.RecordsOnPage = int(math.Mod(float64(count), float64(perPage)))

		if info.RecordsOnPage == 0 && count > 0 {
			info.RecordsOnPage = perPage
		}

	}

	return info, nil
}
