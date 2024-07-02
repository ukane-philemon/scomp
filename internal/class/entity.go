package class

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
	idKey     = "_id"
	reportKey = "report"
	nameKey   = "name"
)

type Class struct {
	ID            string       `json:"_id" bson:"_id"`
	Name          string       `json:"name" bson:"name"`
	Subjects      []*Subject   `json:"subjects" bson:"subjects"`
	Report        *ClassReport `json:"report" bson:"report"` // nil until a report is generated
	CreatedAt     string       `json:"createdAt" bson:"createdAt"`
	LastUpdatedAt string       `json:"lastUpdatedAt" bson:"lastUpdatedAt"`
}

type Subject struct {
	Name     string `json:"name" bson:"name"`
	MaxScore int    `json:"maxScore" bson:"maxScore"`
}

type ClassReport struct {
	TotalStudents                   int    `json:"totalStudents" bson:"totalStudents"`
	HighestStudentScore             int    `json:"highestStudentScore" bson:"highestStudentScore"`
	HighestStudentScoreAsPercentage string `json:"highestStudentScoreAsPercentage" bson:"highestStudentScoreAsPercentage"`
	LowestStudentScore              int    `json:"lowestStudentScore" bson:"lowestStudentScore"`
	LowestStudentScoreAsPercentage  string `json:"lowestStudentScoreAsPercentage" bson:"lowestStudentScoreAsPercentage"`
	GeneratedAt                     string `json:"generatedAt" bson:"generatedAt"`
}

type ClassRepository struct {
	ctx             context.Context
	classCollection *mongo.Collection
}

// NewRepository creates a new instance of *ClassRepository.
func NewRepository(ctx context.Context, db *mongo.Database) (Repository, error) {
	classCollectionIndex := mongo.IndexModel{
		Keys: bson.D{{
			Key:   nameKey,
			Value: 1,
		}},
		Options: options.Index().SetUnique(true),
	}

	// Create a unique index on the class collection.
	classCollection := db.Collection("classes")
	_, err := classCollection.Indexes().CreateOne(ctx, classCollectionIndex)
	if err != nil {
		return nil, err
	}

	return &ClassRepository{
		ctx:             ctx,
		classCollection: classCollection,
	}, nil
}

// Create creates a new class in the database. Returns
// db.ErrorInvalidRequest is the provided class name matches any record in the
// database.
// Implements Repository.
func (cr *ClassRepository) Create(className string, subjects []*Subject) (string, error) {
	if className == "" {
		return "", fmt.Errorf("%w: missing class name", db.ErrorInvalidRequest)
	}

	if len(subjects) != db.RequiredClassSubjects {
		return "", fmt.Errorf("%w: %d class subjects are required to create a class", db.ErrorInvalidRequest, db.RequiredClassSubjects)
	}

	for index, subject := range subjects {
		if subject.Name == "" {
			return "", fmt.Errorf("%w: subject %d is missing subject name", db.ErrorInvalidRequest, index+1)
		}

		if subject.MaxScore < 1 {
			return "", fmt.Errorf("%w: subject %s has an invalid max score %d", db.ErrorInvalidRequest, subject.Name, subject.MaxScore)
		}
	}

	nowUnix := time.Now().Unix()
	classInfo := &Class{
		ID:            primitive.NewObjectID().Hex(),
		Name:          className,
		Subjects:      subjects,
		CreatedAt:     fmt.Sprint(nowUnix),
		LastUpdatedAt: fmt.Sprint(nowUnix),
	}

	res, err := cr.classCollection.InsertOne(cr.ctx, classInfo)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return "", fmt.Errorf("%w: class name %s already exists", db.ErrorInvalidRequest, className)
		}
		return "", fmt.Errorf("classCollection.InsertOne error: %w", err)
	}

	return res.InsertedID.(string), nil
}

// Class returns information for the class that match the provided classID.
// Implements Repository.
func (cr *ClassRepository) Class(classID string) (*Class, error) {
	classFilter, err := classFilter(classID)
	if err != nil {
		return nil, err
	}

	var cInfo *Class
	err = cr.classCollection.FindOne(cr.ctx, classFilter).Decode(&cInfo)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%w: no record found for class with ID %s", db.ErrorInvalidRequest, classID)
		}
		return nil, fmt.Errorf("classCollection.FindOne error: %w", err)
	}

	return cInfo, nil
}

// Classes returns information for all the classes in the database.
// Implements Repository.
func (cr *ClassRepository) Classes(hasReport *bool) ([]*Class, error) {
	filter := bson.M{}
	if hasReport != nil {
		if *hasReport {
			filter = bson.M{reportKey: bson.M{"$exists": true, "$ne": nil}}
		} else {
			filter = bson.M{reportKey: nil}
		}
	}

	cur, err := cr.classCollection.Find(cr.ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("cr.classCollection.Find error: %w", err)
	}

	var classes []*Class
	return classes, cur.All(cr.ctx, &classes)
}

// Exists checks if classID exists.
// Implements Repository.
func (cr *ClassRepository) Exists(classID string) (bool, error) {
	classFilter, err := classFilter(classID)
	if err != nil {
		return false, err
	}

	nClass, err := cr.classCollection.CountDocuments(cr.ctx, classFilter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("classCollection.CountDocuments error: %w", err)
	}

	return nClass > 0, nil
}

// SaveClassReport saves a newly generated class report for the class that match
// the provided classID.
// Implements Repository.
func (cr *ClassRepository) SaveClassReport(classID string, report *ClassReport) error {
	classFilter, err := classFilter(classID)
	if err != nil {
		return err
	}

	res, err := cr.classCollection.UpdateOne(cr.ctx, classFilter, bson.M{"$set": bson.M{reportKey: report}}, options.Update().SetUpsert(false))
	if err != nil {
		return fmt.Errorf("classCollection.UpdateOne error: %w", err)
	}

	if res.ModifiedCount == 0 {
		return fmt.Errorf("%w: report for class with ID %s was not updated", db.ErrorInvalidRequest, classID)
	}

	return nil
}

func classFilter(classID string) (bson.M, error) {
	if classID == "" {
		return nil, fmt.Errorf("%w: missing classID", db.ErrorInvalidRequest)
	}

	return bson.M{idKey: classID}, nil
}
