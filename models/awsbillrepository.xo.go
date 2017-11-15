// Package models contains the types for schema 'trackit'.
package models

// Code generated by xo. DO NOT EDIT.

import (
	"errors"
)

// AwsBillRepository represents a row from 'trackit.aws_bill_repository'.
type AwsBillRepository struct {
	ID           int    `json:"id"`             // id
	AwsAccountID int    `json:"aws_account_id"` // aws_account_id
	Bucket       string `json:"bucket"`         // bucket
	Prefix       string `json:"prefix"`         // prefix

	// xo fields
	_exists, _deleted bool
}

// Exists determines if the AwsBillRepository exists in the database.
func (abr *AwsBillRepository) Exists() bool {
	return abr._exists
}

// Deleted provides information if the AwsBillRepository has been deleted from the database.
func (abr *AwsBillRepository) Deleted() bool {
	return abr._deleted
}

// Insert inserts the AwsBillRepository to the database.
func (abr *AwsBillRepository) Insert(db XODB) error {
	var err error

	// if already exist, bail
	if abr._exists {
		return errors.New("insert failed: already exists")
	}

	// sql insert query, primary key provided by autoincrement
	const sqlstr = `INSERT INTO trackit.aws_bill_repository (` +
		`aws_account_id, bucket, prefix` +
		`) VALUES (` +
		`?, ?, ?` +
		`)`

	// run query
	XOLog(sqlstr, abr.AwsAccountID, abr.Bucket, abr.Prefix)
	res, err := db.Exec(sqlstr, abr.AwsAccountID, abr.Bucket, abr.Prefix)
	if err != nil {
		return err
	}

	// retrieve id
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// set primary key and existence
	abr.ID = int(id)
	abr._exists = true

	return nil
}

// Update updates the AwsBillRepository in the database.
func (abr *AwsBillRepository) Update(db XODB) error {
	var err error

	// if doesn't exist, bail
	if !abr._exists {
		return errors.New("update failed: does not exist")
	}

	// if deleted, bail
	if abr._deleted {
		return errors.New("update failed: marked for deletion")
	}

	// sql query
	const sqlstr = `UPDATE trackit.aws_bill_repository SET ` +
		`aws_account_id = ?, bucket = ?, prefix = ?` +
		` WHERE id = ?`

	// run query
	XOLog(sqlstr, abr.AwsAccountID, abr.Bucket, abr.Prefix, abr.ID)
	_, err = db.Exec(sqlstr, abr.AwsAccountID, abr.Bucket, abr.Prefix, abr.ID)
	return err
}

// Save saves the AwsBillRepository to the database.
func (abr *AwsBillRepository) Save(db XODB) error {
	if abr.Exists() {
		return abr.Update(db)
	}

	return abr.Insert(db)
}

// Delete deletes the AwsBillRepository from the database.
func (abr *AwsBillRepository) Delete(db XODB) error {
	var err error

	// if doesn't exist, bail
	if !abr._exists {
		return nil
	}

	// if deleted, bail
	if abr._deleted {
		return nil
	}

	// sql query
	const sqlstr = `DELETE FROM trackit.aws_bill_repository WHERE id = ?`

	// run query
	XOLog(sqlstr, abr.ID)
	_, err = db.Exec(sqlstr, abr.ID)
	if err != nil {
		return err
	}

	// set deleted
	abr._deleted = true

	return nil
}

// AwsAccount returns the AwsAccount associated with the AwsBillRepository's AwsAccountID (aws_account_id).
//
// Generated from foreign key 'aws_bill_repository_ibfk_1'.
func (abr *AwsBillRepository) AwsAccount(db XODB) (*AwsAccount, error) {
	return AwsAccountByID(db, abr.AwsAccountID)
}

// AwsBillRepositoriesByAwsAccountID retrieves a row from 'trackit.aws_bill_repository' as a AwsBillRepository.
//
// Generated from index 'aws_account_id'.
func AwsBillRepositoriesByAwsAccountID(db XODB, awsAccountID int) ([]*AwsBillRepository, error) {
	var err error

	// sql query
	const sqlstr = `SELECT ` +
		`id, aws_account_id, bucket, prefix ` +
		`FROM trackit.aws_bill_repository ` +
		`WHERE aws_account_id = ?`

	// run query
	XOLog(sqlstr, awsAccountID)
	q, err := db.Query(sqlstr, awsAccountID)
	if err != nil {
		return nil, err
	}
	defer q.Close()

	// load results
	res := []*AwsBillRepository{}
	for q.Next() {
		abr := AwsBillRepository{
			_exists: true,
		}

		// scan
		err = q.Scan(&abr.ID, &abr.AwsAccountID, &abr.Bucket, &abr.Prefix)
		if err != nil {
			return nil, err
		}

		res = append(res, &abr)
	}

	return res, nil
}

// AwsBillRepositoryByID retrieves a row from 'trackit.aws_bill_repository' as a AwsBillRepository.
//
// Generated from index 'aws_bill_repository_id_pkey'.
func AwsBillRepositoryByID(db XODB, id int) (*AwsBillRepository, error) {
	var err error

	// sql query
	const sqlstr = `SELECT ` +
		`id, aws_account_id, bucket, prefix ` +
		`FROM trackit.aws_bill_repository ` +
		`WHERE id = ?`

	// run query
	XOLog(sqlstr, id)
	abr := AwsBillRepository{
		_exists: true,
	}

	err = db.QueryRow(sqlstr, id).Scan(&abr.ID, &abr.AwsAccountID, &abr.Bucket, &abr.Prefix)
	if err != nil {
		return nil, err
	}

	return &abr, nil
}
