package student

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ukane-philemon/scomp/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	idKey      = "_id"
	nameKey    = "name"
	classIDKey = "classID"
	reportKey  = "report"
)

type Student struct {
	ID        string  `json:"_id" bson:"_id"`
	Name      string  `json:"name" bson:"name"`
	ClassID   string  `json:"classID" bson:"classID"`
	Report    *Report `json:"report" bson:"report"`
	CreatedAt string  `json:"createdAt" bson:"createdAt"`
}

type Report struct {
	Subjects    []*SubjectReport    `json:"subjects" bson:"subjects"`
	Class       *StudentClassReport `json:"class" bson:"class"`
	GeneratedAt string              `json:"generatedAt" bson:"generatedAt"`
}

type StudentClassReport struct {
	Position             int    `json:"position" bson:"position"`
	Grade                string `json:"grade" bson:"grade"`
	TotalScore           int    `json:"totalScore" bson:"totalScore"`
	TotalScorePercentage string `json:"totalScorePercentage" bson:"totalScorePercentage"`
}

type SubjectReport struct {
	*SubjectScore `bson:"inline"`
	Grade         string `json:"grade,omitempty" bson:"grade"`
	Position      int    `json:"position,omitempty" bson:"position"`
}

type SubjectScore struct {
	Name  string `json:"name" bson:"name"`
	Score int    `json:"score" bson:"score"`
}

// StudentRepository implements Repository.
type StudentRepository struct {
	ctx               context.Context
	studentCollection *mongo.Collection
}

// NewRepository creates a new instance of *StudentRepository.
func NewRepository(ctx context.Context, db *mongo.Database) (Repository, error) {
	studentCollectionIndex := mongo.IndexModel{
		Keys: bson.D{{
			Key:   nameKey,
			Value: 1,
		}, {
			Key:   classIDKey,
			Value: 1,
		}},
		Options: options.Index().SetUnique(true),
	}

	// Create a unique index on the class collection.
	classCollection := db.Collection("students")
	_, err := classCollection.Indexes().CreateOne(ctx, studentCollectionIndex)
	if err != nil {
		return nil, err
	}

	return &StudentRepository{
		ctx:               ctx,
		studentCollection: classCollection,
	}, nil
}

// Create adds a students record. Returns db.ErrorInvalidRequest if studentName
// already exists for classID.
// Implements Repository.
func (sr *StudentRepository) Create(classID string, studentName string, subjectScores []*SubjectScore) (string, error) {
	if classID == "" || studentName == "" || len(subjectScores) == 0 {
		return "", fmt.Errorf("%w: missing required argument(s)", db.ErrorInvalidRequest)
	}

	if len(subjectScores) != db.RequiredClassSubjects {
		return "", fmt.Errorf("%w: %d class subjects are required to save a student's record", db.ErrorInvalidRequest, db.RequiredClassSubjects)
	}

	student := &Student{
		ID:        primitive.NewObjectID().Hex(),
		Name:      studentName,
		ClassID:   classID,
		Report:    new(Report),
		CreatedAt: fmt.Sprint(time.Now().Unix()),
	}

	for index, subject := range subjectScores {
		if subject.Name == "" {
			return "", fmt.Errorf("%w: student subject %d is missing subject name", db.ErrorInvalidRequest, index+1)
		}

		if subject.Score < 0 {
			return "", fmt.Errorf("%w: subject %s has an invalid score %d", db.ErrorInvalidRequest, subject.Name, subject.Score)
		}

		student.Report.Subjects = append(student.Report.Subjects, &SubjectReport{
			SubjectScore: subject,
		})
	}

	// Create student record.
	res, err := sr.studentCollection.InsertOne(sr.ctx, student)
	if err != nil {
		return "", fmt.Errorf("studentCollection.InsertOne error: %w", err)
	}

	return res.InsertedID.(string), nil
}

// Student returns the students that match provided arguments.
// Implements Repository.
func (sr *StudentRepository) Student(classID string, studentID string) (*Student, error) {
	if classID == "" || studentID == "" {
		return nil, fmt.Errorf("%w: missing required argument(s)", db.ErrorInvalidRequest)
	}

	var student *Student
	studentFilter := bson.M{idKey: studentID, classIDKey: classID}
	err := sr.studentCollection.FindOne(sr.ctx, studentFilter).Decode(&student)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%w: no record found for student with ID %s", db.ErrorInvalidRequest, studentID)
		}
		return nil, fmt.Errorf("studentCollection.FindOne error: %w", err)
	}

	return student, nil
}

// Students returns all the students that match the provided classID.
// Implements Repository.
func (sr *StudentRepository) Students(classID string) ([]*Student, error) {
	if classID == "" {
		return nil, fmt.Errorf("%w: missing classID", db.ErrorInvalidRequest)
	}

	cur, err := sr.studentCollection.Find(sr.ctx, bson.M{classIDKey: classID})
	if err != nil {
		return nil, fmt.Errorf("studentCollection.Find error: %w", err)
	}

	var students []*Student
	return students, cur.All(sr.ctx, &students)
}

// StudentScores returns a map of student ID to their subject scores.
// Implements Repository.
func (sr *StudentRepository) StudentScores(classID string) (map[string][]*SubjectScore, error) {
	if classID == "" {
		return nil, fmt.Errorf("%w: missing classID", db.ErrorInvalidRequest)
	}

	cur, err := sr.studentCollection.Find(sr.ctx, bson.M{classIDKey: classID})
	if err != nil {
		return nil, fmt.Errorf("studentCollection.Find error: %w", err)
	}

	var students []*Student
	err = cur.All(sr.ctx, &students)
	if err != nil {
		return nil, fmt.Errorf("failed to decode student information: %w", err)
	}

	studentsMap := make(map[string][]*SubjectScore, len(students))
	for _, student := range students {
		studentsMap[student.ID] = studentSubjectScores(student.Report.Subjects)
	}

	return studentsMap, nil
}

func studentSubjectScores(report []*SubjectReport) []*SubjectScore {
	var scores []*SubjectScore
	for _, r := range report {
		scores = append(scores, r.SubjectScore)
	}
	return scores
}

// SaveStudentReports saves the students report specified.
// Implements Repository.
func (sr *StudentRepository) SaveStudentReports(reports map[string]*Report) error {
	session, err := sr.studentCollection.Database().Client().StartSession()
	if err != nil {
		return fmt.Errorf("Client().StartSession() error: %w", err)
	}
	defer session.EndSession(sr.ctx)

	saveStudentReportFn := func(ctx mongo.SessionContext) (interface{}, error) {
		for studentID, report := range reports {
			update := bson.M{"$set": bson.M{reportKey: report}}
			res, err := sr.studentCollection.UpdateOne(ctx, bson.M{idKey: studentID}, update, options.Update().SetUpsert(false))
			if err != nil {
				return nil, fmt.Errorf("studentCollection.UpdateOne error: %w", err)
			}

			if res.ModifiedCount == 0 {
				return nil, fmt.Errorf("student with ID %s was not updated", studentID)
			}
		}

		return nil, nil
	}

	_, err = session.WithTransaction(sr.ctx, saveStudentReportFn)
	if err != nil {
		return err
	}

	return nil
}
