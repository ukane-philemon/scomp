package graph

import (
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/ukane-philemon/scomp/graph/model"
	"github.com/ukane-philemon/scomp/internal/jwt"
)

type Resolver struct {
	wg         sync.WaitGroup
	db         ClassDatabase
	JWTManager *jwt.Manager
}

// NewResolver creates and returns a new instance of *Resolver.
func NewResolver(db ClassDatabase) (*Resolver, error) {
	jwtManager, err := jwt.NewJWTManager()
	if err != nil {
		return nil, fmt.Errorf("jwt.NewJWTManager error: %w", err)
	}

	return &Resolver{
		db:         db,
		JWTManager: jwtManager,
	}, nil
}

// Wait waits for all pending asynchronous activities to finish.
func (r *Resolver) Wait() {
	r.wg.Wait()
}

type studentScoreReport struct {
	studentID    string
	subjectScore int
}

type subjectScoreInfo struct {
	studentScores []*studentScoreReport
	maxScore      int
}

// computeClassReport generates a report for a class.
func (r *Resolver) computeClassReport(classID string, classSubjects map[string]*model.Subject, studentsInfo map[string]*model.StudentSubjectScore) {
	var totalMaxSubjectsScore int
	subjectScoreMap := make(map[string]*subjectScoreInfo, len(classSubjects))
	for subjectName, subjectInfo := range classSubjects {
		totalMaxSubjectsScore += subjectInfo.MaxScore
		subjectScoreMap[subjectName] = &subjectScoreInfo{
			maxScore: subjectInfo.MaxScore,
		}
	}

	// studentRecordsMap is a map of studentID to student record.
	studentRecordsMap := make(map[string]*model.StudentRecord, len(studentsInfo))

	// Compute max scores and subject scores for all students.
	for studentName, studentInfo := range studentsInfo {
		var totalScore int
		for _, subject := range studentInfo.SubjectScores {
			totalScore += subject.Score
			subjectScoreMap[subject.Name].studentScores = append(subjectScoreMap[subject.Name].studentScores, &studentScoreReport{
				studentID:    studentInfo.StudentID,
				subjectScore: subject.Score,
			})
		}

		studentRecordsMap[studentInfo.StudentID] = &model.StudentRecord{
			ID:   studentInfo.StudentID,
			Name: studentName,
			Report: &model.StudentReport{
				TotalScore:           totalScore,
				TotalScorePercentage: float64(totalScore/totalMaxSubjectsScore) * 100,
			},
		}
	}

	// Set student subject position an grade them.
	for subjectName, subject := range subjectScoreMap {
		// Sort according to highest subject scores.
		sort.SliceStable(subject.studentScores, func(i, j int) bool {
			return subject.studentScores[i].subjectScore > subject.studentScores[j].subjectScore
		})

		// Set student position and grade them.
		for positionIndex, record := range subject.studentScores {
			subjectScorePercentage := float64(record.subjectScore/subject.maxScore) * 100
			studentRecordsMap[record.studentID].Report.SubjectReport = append(studentRecordsMap[record.studentID].Report.SubjectReport, &model.StudentSubjectReport{
				Name:     subjectName,
				Score:    record.subjectScore,
				Grade:    gradeTotalScorePercentage(subjectScorePercentage),
				Position: positionIndex + 1,
			})
		}
	}

	var studentRecords []*model.StudentRecord
	for _, record := range studentRecordsMap {
		studentRecords = append(studentRecords, record)
	}

	// Sort according to highest total scores.
	sort.SliceStable(studentRecords, func(i, j int) bool {
		return studentRecords[i].Report.TotalScore > studentRecords[j].Report.TotalScore
	})

	// scoreMap is a map of student total scores to a student position and is
	// used to ensure students with the same total scores get the same position
	scoreMap := make(map[int]int)

	classReport := new(model.ClassReport)

	// Set student position and grade them.
	for positionIndex, record := range studentRecords {
		report := record.Report
		studentPosition, foundScore := scoreMap[report.TotalScore]
		if !foundScore {
			studentPosition = positionIndex + 1
			scoreMap[report.TotalScore] = studentPosition
		}

		studentRecords[positionIndex].Report.ClassPosition = studentPosition
		studentRecords[positionIndex].Report.ClassGrade = gradeTotalScorePercentage(report.TotalScorePercentage)

		if report.TotalScore > classReport.HighestStudentScore {
			classReport.HighestStudentScore = report.TotalScore
		} else if report.TotalScore < classReport.LowestStudentScore {
			classReport.LowestStudentScore = report.TotalScore
		}
	}

	classReport.HighestStudentScoreAsPercentage = float64(classReport.HighestStudentScore/totalMaxSubjectsScore) * 100
	classReport.LowestStudentScoreAsPercentage = float64(classReport.LowestStudentScore/totalMaxSubjectsScore) * 100

	err := r.db.SaveClassReport(classID, classReport, studentRecords)
	if err != nil {
		log.Printf("\nSERVER ERROR: %v", err.Error())
	}
}

func gradeTotalScorePercentage(totalStudentScorePercentage float64) string {
	switch {
	case totalStudentScorePercentage > 69:
		return "Excellent"
	case totalStudentScorePercentage > 59:
		return "Good"
	case totalStudentScorePercentage > 49:
		return "Fair"
	case totalStudentScorePercentage > 44:
		return "Pass"
	case totalStudentScorePercentage > 40:
		return "Pass"
	default:
		return "Fail"
	}
}
