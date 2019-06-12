// Code generated by gogen.go; DO NOT EDIT.

package model

import (
	"crypto/md5"
	"encoding/hex"
)

const (
	Class_AccessToken      = "accessToken"
	Class_Capacity         = "capacity"
	Class_Description      = "description"
	Class_EvaluationCodes  = "evaluationCodes"
	Class_ImportHash       = "importHash"
	Class_InstructorEmails = "instructorEmails"
	Class_InstructorNames  = "instructorNames"
	Class_Length           = "length"
	Class_Location         = "location"
	Class_New              = "new"
	Class_Programs         = "programs"
	Class_Responsibility   = "responsibility"
	Class_SpreadsheetRow   = "spreadsheetRow"
	Class_Title            = "title"
	Class_TitleNote        = "titleNote"
)

func (x *Class) CopyImportFieldsTo(y *Class) {
	y.AccessToken = x.AccessToken
	y.Capacity = x.Capacity
	y.Description = x.Description
	y.EvaluationCodes = x.EvaluationCodes
	y.InstructorEmails = x.InstructorEmails
	y.InstructorNames = x.InstructorNames
	y.Length = x.Length
	y.Location = x.Location
	y.New = x.New
	y.Number = x.Number
	y.Programs = x.Programs
	y.Responsibility = x.Responsibility
	y.SpreadsheetRow = x.SpreadsheetRow
	y.Title = x.Title
	y.TitleNote = x.TitleNote
}

func (x *Class) EqualImportFields(y *Class) bool {
	return x.AccessToken == y.AccessToken &&
		x.Capacity == y.Capacity &&
		x.Description == y.Description &&
		equalStringSlice(x.EvaluationCodes, y.EvaluationCodes) &&
		equalStringSlice(x.InstructorEmails, y.InstructorEmails) &&
		equalStringSlice(x.InstructorNames, y.InstructorNames) &&
		x.Length == y.Length &&
		x.Location == y.Location &&
		x.New == y.New &&
		x.Number == y.Number &&
		x.Programs == y.Programs &&
		x.Responsibility == y.Responsibility &&
		x.SpreadsheetRow == y.SpreadsheetRow &&
		x.Title == y.Title &&
		x.TitleNote == y.TitleNote
}

func (x *Class) HashImportFields() string {
	h := md5.New()
	hashValue(h, "bc11beba53e3b91809849e58d8a81de1")
	hashValue(h, x.AccessToken)
	hashValue(h, x.Capacity)
	hashValue(h, x.Description)
	hashValue(h, x.EvaluationCodes)
	hashValue(h, x.InstructorEmails)
	hashValue(h, x.InstructorNames)
	hashValue(h, x.Length)
	hashValue(h, x.Location)
	hashValue(h, x.New)
	hashValue(h, x.Number)
	hashValue(h, x.Programs)
	hashValue(h, x.Responsibility)
	hashValue(h, x.SpreadsheetRow)
	hashValue(h, x.Title)
	hashValue(h, x.TitleNote)
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:])
}
