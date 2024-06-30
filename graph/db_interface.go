package graph

import (
	"context"

	"github.com/ukane-philemon/scomp/graph/model"
)

type ClassDatabase interface {
	// CreateAdminAccount creates a new admin with the provided username and
	// password. An ErrorInvalidRequest will be returned is the username already
	// exists.
	CreateAdminAccount(username, password string) error
	// Login checks that the provided username and password matches a record in
	// the database and are correct. Returns db.ErrorInvalidRequest if the password
	// or username does not match any record.
	Login(username, password string) (*model.Admin, error)
	// CreateClass creates a new class in the database. Returns
	// db.ErrorInvalidRequest is the provided class name matches any record in the
	// database.
	CreateClass(class *model.NewClass) (string, error)
	// AddStudentRecordToClass adds a students record to an existing class.
	// Returns db.ErrorInvalidRequest if a class report has already been
	// generated for this class or student already exists.
	AddStudentRecordToClass(classID string, studentRecord *model.StudentRecordInput) (string, error)
	// SaveClassReport saves a generated class report. Returns
	// db.ErrorInvalidRequest if no students exists in the class that match the
	// provided classID or a report has been generated for this class.
	SaveClassReport(classID string, classReport *model.ClassReport, studentsReport []*model.StudentRecord) error
	// ClassInfo retrieves the class information that matches the provided
	// classID.
	ClassInfo(classID string) (*model.ClassInfo, error)
	// ClassInfoForReport returns the class details required to generate a
	// report. Returns db.ErrorInvalidRequest if or a report has been generated
	// for this class.
	ClassInfoForReport(classID string) (map[string]*model.Subject, map[string][]*model.StudentSubjectScoreInput, error)
	// Classes returns information for all classes from the database. Set
	// hasReport to true to return only classes that have a report.
	Classes(hasReport *bool) ([]*model.ClassInfo, error)
	// StudentRecord retrieves the student record that match the specified
	// parameters. Returns db.ErrorInvalidRequest if no student is found.
	StudentRecord(classID string, studentID string) (*model.StudentRecord, error)
	// Shutdown gracefully disconnects the database after the server is
	// shutdown.
	Shutdown(ctx context.Context) error
}
