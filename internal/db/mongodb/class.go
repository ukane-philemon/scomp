package mongodb

import (
	"errors"
	"fmt"
	"time"

	"github.com/ukane-philemon/scomp/graph/model"
	"github.com/ukane-philemon/scomp/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// minStudentsForReport is the minimum number of students required to generate a
// class report.
const minStudentsForReport = 2

// CreateClass creates a new class in the database. Returns
// db.ErrorInvalidRequest is the provided class name matches any record in the
// database.
func (mdb *MongoDB) CreateClass(class *model.NewClass) (string, error) {
	if class.Name == "" {
		return "", fmt.Errorf("%w: missing class name", db.ErrorInvalidRequest)
	}

	if len(class.ClassSubjects) != db.RequiredClassSubjects {
		return "", fmt.Errorf("%w: %d class subjects are required to create a class", db.ErrorInvalidRequest, db.RequiredClassSubjects)
	}

	subjectMap := make(map[string]*model.Subject)
	for index, subject := range class.ClassSubjects {
		if subject.Name == "" {
			return "", fmt.Errorf("%w: subject %d is missing subject name", db.ErrorInvalidRequest, index+1)
		}

		if subject.MaxScore < 1 {
			return "", fmt.Errorf("%w: subject %s has an invalid max score %d", db.ErrorInvalidRequest, subject.Name, subject.MaxScore)
		}

		subjectMap[subject.Name] = (*model.Subject)(subject)
	}

	nowUnix := time.Now().Unix()
	classInfo := &dbClassInfo{
		ID:                    primitive.NewObjectID(),
		Name:                  class.Name,
		Subjects:              subjectMap,
		StudentsSubjectRecord: make(map[string]*model.StudentSubjectScore),
		CreatedAt:             nowUnix,
		LastUpdatedAt:         nowUnix,
	}

	res, err := mdb.classCollection.InsertOne(mdb.ctx, classInfo)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return "", fmt.Errorf("%w: class name %s already exists", db.ErrorInvalidRequest, class.Name)
		}
		return "", fmt.Errorf("classCollection.InsertOne error: %w", err)
	}

	return res.InsertedID.(primitive.ObjectID).Hex(), nil
}

// AddStudentRecordToClass adds a students record to an existing class. Returns
// db.ErrorInvalidRequest if a class report has already been generated for this
// class or student already exists.
func (mdb *MongoDB) AddStudentRecordToClass(classID string, studentRecord *model.StudentRecordInput) (string, error) {
	if classID == "" || studentRecord == nil || studentRecord.Name == "" {
		return "", fmt.Errorf("%w: missing required argument(s)", db.ErrorInvalidRequest)
	}

	if len(studentRecord.SubjectScores) != db.RequiredClassSubjects {
		return "", fmt.Errorf("%w: %d class subjects are required to save a student's report", db.ErrorInvalidRequest, db.RequiredClassSubjects)
	}

	dbClassID, err := primitive.ObjectIDFromHex(classID)
	if err != nil {
		return "", fmt.Errorf("%w: invalid classID %s", db.ErrorInvalidRequest, classID)
	}

	opts := options.FindOne().SetProjection(bson.M{
		dbIDKey:        1,
		subjectsKey:    1,
		classReportKey: 1,
		mapKey(studentSubjectsRecordKey, studentRecord.Name): 1,
	})

	// Retrieve class information.
	var dbClass *dbClassInfo
	classFilter := bson.M{dbIDKey: dbClassID}
	err = mdb.classCollection.FindOne(mdb.ctx, classFilter, opts).Decode(&dbClass)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", fmt.Errorf("%w: class with ID %s does not exist", db.ErrorInvalidRequest, classID)
		}
		return "", fmt.Errorf("classCollection.FindOne error: %w", err)
	}

	if dbClass.ClassReport != nil {
		return "", fmt.Errorf("%w: cannot add student record to class with a finalized report", db.ErrorInvalidRequest)
	}

	_, studentExists := dbClass.StudentsSubjectRecord[studentRecord.Name]
	if studentExists {
		return "", fmt.Errorf("%w: student name %s has already been added to this class", db.ErrorInvalidRequest, studentRecord.Name)
	}

	for index, subject := range studentRecord.SubjectScores {
		if subject.Name == "" {
			return "", fmt.Errorf("%w: student subject %d is missing subject name", db.ErrorInvalidRequest, index+1)
		}

		if subject.Score < 0 {
			return "", fmt.Errorf("%w: subject %s has an invalid max score %d", db.ErrorInvalidRequest, subject.Name, subject.Score)
		}

		classSubjectInfo, found := dbClass.Subjects[subject.Name]
		if !found {
			return "", fmt.Errorf("%w: subject name %s does not exist, check spelling as subject names are case sensitive.",
				db.ErrorInvalidRequest, subject.Name)
		}

		if subject.Score > classSubjectInfo.MaxScore {
			return "", fmt.Errorf("%w: student score (%d) for subject %s exceeds the maximum score (%d) for this subject",
				db.ErrorInvalidRequest, subject.Score, subject.Name, classSubjectInfo.MaxScore)
		}
	}

	addStudentFn := func(ctx mongo.SessionContext) (interface{}, error) {
		// Create student record.
		dbStudentRecord := &dbStudentRecord{
			ID:      primitive.NewObjectID(),
			Name:    studentRecord.Name,
			ClassID: classID,
		}
		res, err := mdb.studentCollection.InsertOne(ctx, dbStudentRecord)
		if err != nil {
			return "", fmt.Errorf("studentCollection.InsertOne error: %w", err)
		}

		studentIDStr := res.InsertedID.(primitive.ObjectID).Hex()

		// Update the class record with student information.
		classCollectionUpdate := bson.M{actionSet: bson.M{
			mapKey(studentSubjectsRecordKey, studentRecord.Name): &model.StudentSubjectScore{
				StudentID:     studentIDStr,
				SubjectScores: convertToSubjectScoreInput(studentRecord.SubjectScores),
			},
			lastUpdatedAtKey: time.Now().Unix(),
		}}
		_, err = mdb.classCollection.UpdateOne(ctx, classFilter, classCollectionUpdate)
		if err != nil {
			return "", fmt.Errorf("adminCollection.UpdateOne error: %w", err)
		}

		return studentIDStr, nil
	}

	studentID, err := mdb.withSession(addStudentFn)
	if err != nil {
		return "", err
	}

	return studentID.(string), nil
}

// SaveClassReport saves a generated class report. Returns
// db.ErrorInvalidRequest if no students exists in the class that match the
// provided classID or a report has been generated for this class.
func (mdb *MongoDB) SaveClassReport(classID string, classReport *model.ClassReport, studentsReport []*model.StudentRecord) error {
	if classID == "" || classReport == nil || len(studentsReport) == 0 {
		return fmt.Errorf("%w: missing required argument(s)", db.ErrorInvalidRequest)
	}

	dbClassID, err := primitive.ObjectIDFromHex(classID)
	if err != nil {
		return fmt.Errorf("%w: invalid classID %s", db.ErrorInvalidRequest, classID)
	}

	var cInfo *dbClassInfo
	classFilter := bson.M{dbIDKey: dbClassID}
	err = mdb.classCollection.FindOne(mdb.ctx, classFilter).Decode(&cInfo)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("%w: no record found for class with ID %s", db.ErrorInvalidRequest, classID)
		}
		return fmt.Errorf("mdb.classCollection.FindOne error: %w", err)
	}

	if cInfo.ClassReport != nil {
		return fmt.Errorf("%w: a report has already been generated for this class", db.ErrorInvalidRequest)
	}

	saveReportFn := func(ctx mongo.SessionContext) (interface{}, error) {
		classCollectionUpdate := bson.M{actionSet: bson.M{
			classReportKey:   classReport,
			lastUpdatedAtKey: time.Now().Unix(),
		}}
		res, err := mdb.classCollection.UpdateOne(mdb.ctx, classFilter, classCollectionUpdate, options.Update().SetUpsert(false))
		if err != nil {
			return nil, fmt.Errorf("classCollection.UpdateOne error: %w", err)
		}

		if res.ModifiedCount == 0 {
			return nil, fmt.Errorf("%w: no record found for class with ID %s", db.ErrorInvalidRequest, classID)
		}

		for _, studentRecord := range studentsReport {
			studentID, err := primitive.ObjectIDFromHex(studentRecord.ID)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid studentID %s", db.ErrorInvalidRequest, studentID)
			}

			studentFilter := bson.M{
				dbIDKey: studentID,
				classID: classID,
			}

			studentCollectionUpdate := bson.M{actionSet: bson.M{
				studentReportKey: studentRecord.Report,
			}}
			res, err := mdb.studentCollection.UpdateOne(mdb.ctx, studentFilter, studentCollectionUpdate, options.Update().SetUpsert(false))
			if err != nil {
				return nil, fmt.Errorf("studentCollection.UpdateOne error: %w", err)
			}

			if res.ModifiedCount == 0 {
				return nil, fmt.Errorf("%w: no record found for student (ID: %s, Name: %s)",
					db.ErrorInvalidRequest, studentRecord.ID, studentRecord.Name)
			}
		}

		return nil, nil
	}

	_, err = mdb.withSession(saveReportFn)
	if err != nil {
		return err
	}

	return nil
}

// ClassInfo retrieves the class information that matches the provided classID.
func (mdb *MongoDB) ClassInfo(classID string) (*model.ClassInfo, error) {
	if classID == "" {
		return nil, fmt.Errorf("%w: missing classID", db.ErrorInvalidRequest)
	}

	dbClassID, err := primitive.ObjectIDFromHex(classID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid classID %s", db.ErrorInvalidRequest, classID)
	}

	classFilter := bson.M{dbIDKey: dbClassID}
	var cInfo *dbClassInfo
	err = mdb.classCollection.FindOne(mdb.ctx, classFilter).Decode(&cInfo)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%w: no record found for class with ID %s", db.ErrorInvalidRequest, classID)
		}
		return nil, fmt.Errorf("mdb.classCollection.FindOne error: %w", err)
	}

	classStudents, err := mdb.classStudents(classID)
	if err != nil {
		return nil, err
	}

	return &model.ClassInfo{
		ID:             cInfo.ID.Hex(),
		Name:           cInfo.Name,
		Subjects:       cInfo.SubjectsToArray(),
		StudentRecords: classStudents,
		ClassReport:    cInfo.ClassReport,
		CreatedAt:      fmt.Sprint(cInfo.CreatedAt),
		LastUpdatedAt:  fmt.Sprint(cInfo.LastUpdatedAt),
	}, nil
}

// ClassInfoForReport returns the class details required to generate a report.
// Returns db.ErrorInvalidRequest if or a report has been generated for this
// class. The second return value us a map of student name to their score
// records.
func (mdb *MongoDB) ClassInfoForReport(classID string) (map[string]*model.Subject, map[string]*model.StudentSubjectScore, error) {
	if classID == "" {
		return nil, nil, fmt.Errorf("%w: missing classID", db.ErrorInvalidRequest)
	}

	dbClassID, err := primitive.ObjectIDFromHex(classID)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: invalid classID %s", db.ErrorInvalidRequest, classID)
	}

	classFilter := bson.M{dbIDKey: dbClassID}
	var cInfo *dbClassInfo
	err = mdb.classCollection.FindOne(mdb.ctx, classFilter).Decode(&cInfo)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil, fmt.Errorf("%w: no record found for class with ID %s", db.ErrorInvalidRequest, classID)
		}
		return nil, nil, fmt.Errorf("mdb.classCollection.FindOne error: %w", err)
	}

	if cInfo.ClassReport != nil {
		return nil, nil, fmt.Errorf("%w: a report has already been generated for this class", db.ErrorInvalidRequest)
	}

	if len(cInfo.StudentsSubjectRecord) < minStudentsForReport {
		return nil, nil, fmt.Errorf("%w: a minimum of %d students is required to generate a report for this class",
			db.ErrorInvalidRequest, minStudentsForReport)
	}

	return cInfo.Subjects, cInfo.StudentsSubjectRecord, nil
}

// StudentRecord retrieves the student record that match the specified
// parameters. Returns db.ErrorInvalidRequest if no student is found.
func (mdb *MongoDB) StudentRecord(classID string, studentID string) (*model.StudentRecord, error) {
	if classID == "" || studentID == "" {
		return nil, fmt.Errorf("%w: missing required argument(s)", db.ErrorInvalidRequest)
	}

	dbStudentID, err := primitive.ObjectIDFromHex(studentID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid student ID %s", db.ErrorInvalidRequest, studentID)
	}

	studentFilter := bson.M{
		dbIDKey:    dbStudentID,
		classIDKey: classID,
	}

	var dbStudent *dbStudentRecord
	err = mdb.studentCollection.FindOne(mdb.ctx, studentFilter).Decode(&dbStudent)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%w: no student match the provided query", db.ErrorInvalidRequest)
		}
		return nil, fmt.Errorf("studentCollection.FindOne error: %w", err)
	}

	return dbStudent.StudentRecord(), nil
}

// Classes returns information for all classes from the database. Set
// hasReport to true to return only classes that have a report.
func (mdb *MongoDB) Classes(hasReport *bool) ([]*model.ClassInfo, error) {
	filter := bson.M{}
	if hasReport != nil {
		if *hasReport {
			filter = bson.M{classReportKey: bson.M{"$exists": true, "$ne": nil}}
		} else {
			filter = bson.M{classReportKey: nil}
		}
	}
	cur, err := mdb.classCollection.Find(mdb.ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("mdb.classCollection.Find error: %w", err)
	}

	var classes []*model.ClassInfo
	for cur.Next(mdb.ctx) {
		var cInfo dbClassInfo
		err = cur.Decode(&cInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to decode classes: %w", err)
		}

		classID := cInfo.ID.Hex()
		classStudents, err := mdb.classStudents(classID)
		if err != nil {
			return nil, err
		}

		classes = append(classes, &model.ClassInfo{
			ID:             cInfo.ID.Hex(),
			Name:           cInfo.Name,
			Subjects:       cInfo.SubjectsToArray(),
			StudentRecords: classStudents,
			ClassReport:    cInfo.ClassReport,
			CreatedAt:      fmt.Sprint(cInfo.CreatedAt),
			LastUpdatedAt:  fmt.Sprint(cInfo.LastUpdatedAt),
		})
	}

	return classes, nil
}

// withSession starts a mongodb session for sessionFn.
func (mdb *MongoDB) withSession(sessionFn func(ctx mongo.SessionContext) (any, error)) (any, error) {
	session, err := mdb.db.Client().StartSession()
	if err != nil {
		return nil, fmt.Errorf("db.Client().StartSession error; %w", err)
	}
	defer session.EndSession(mdb.ctx)

	res, err := session.WithTransaction(mdb.ctx, sessionFn)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// classStudents is a helper method that retrieves all the students in the class
// that match the provided classID.
func (mdb *MongoDB) classStudents(classID string) ([]*model.StudentRecord, error) {
	// Retrieve student record for this class
	studentCur, err := mdb.studentCollection.Find(mdb.ctx, bson.M{classIDKey: classID})
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		return nil, fmt.Errorf("mdb.studentCollection.Find error: %w", err)
	}

	var students []*model.StudentRecord
	for studentCur.Next(mdb.ctx) {
		var sRecord dbStudentRecord
		err = studentCur.Decode(&sRecord)
		if err != nil {
			return nil, fmt.Errorf("failed to decode classes: %w", err)
		}

		students = append(students, sRecord.StudentRecord())
	}

	return students, nil
}

// convertToSubjectScoreInput is a helper function that converts
// []*model.StudentSubjectScoreInput to []*model.SubjectScoreInput.
func convertToSubjectScoreInput(subjects []*model.StudentSubjectScoreInput) []*model.SubjectScoreInput {
	var subjectsScore []*model.SubjectScoreInput
	for _, subject := range subjects {
		subjectsScore = append(subjectsScore, &model.SubjectScoreInput{
			Name:  subject.Name,
			Score: subject.Score,
		})
	}
	return subjectsScore
}
