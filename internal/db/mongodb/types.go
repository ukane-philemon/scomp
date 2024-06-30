package mongodb

import (
	"github.com/ukane-philemon/scomp/graph/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type dbAdmin struct {
	ID        primitive.ObjectID `bson:"_id"`
	Username  string             `bson:"username"`
	Password  string             `bson:"password"`
	CreatedAt int64              `bson:"createdAt"`
}

type dbClassInfo struct {
	ID       primitive.ObjectID        `bson:"_id"`
	Name     string                    `json:"name"`
	Subjects map[string]*model.Subject `json:"subjects"`
	// StudentsSubjectRecord is a map of student name to their subject scores
	// and is used to ensure non-duplicated student for this class.
	StudentsSubjectRecord map[string][]*model.StudentSubjectScoreInput `json:"studentsSubjectRecord"`
	ClassReport           *model.ClassReport                           `json:"classReport"`
	CreatedAt             int64                                        `bson:"createdAt"`
	LastUpdatedAt         int64                                        `bson:"lastUpdatedAt"`
}

func (ci *dbClassInfo) SubjectsToArray() []*model.Subject {
	var subjects []*model.Subject
	for _, subject := range ci.Subjects {
		subjects = append(subjects, subject)
	}
	return subjects
}

type dbStudentRecord struct {
	ID            primitive.ObjectID   `bson:"_id"`
	Name          string               `bson:"name"`
	ClassID       string               `bson:"classID"`
	StudentReport *model.StudentReport `bson:"studentReport"`
}

func (sr *dbStudentRecord) StudentRecord() *model.StudentRecord {
	return &model.StudentRecord{
		ID:     sr.ID.Hex(),
		Name:   sr.Name,
		Report: sr.StudentReport,
	}
}
