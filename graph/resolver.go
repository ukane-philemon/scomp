package graph

import (
	"log"
	"sort"
	"sync"

	"github.com/ukane-philemon/scomp/internal/admin"
	"github.com/ukane-philemon/scomp/internal/auth"
	"github.com/ukane-philemon/scomp/internal/class"
	"github.com/ukane-philemon/scomp/internal/student"
)

type Resolver struct {
	wg sync.WaitGroup

	// Repositories
	AdminRepo   admin.Repository
	ClassRepo   class.Repository
	StudentRepo student.Repository
	AuthRepo    auth.Repository
}

// Wait waits for all pending asynchronous activities to finish.
func (r *Resolver) Wait() {
	r.wg.Wait()
}

type studentSubjectScore struct {
	studentID string
	score     int
}

type subjectScoreInfo struct {
	studentScores []*studentSubjectScore
	maxScore      int
}

type studentReport struct {
	studentID string
	report    *student.Report
}

type subjectReport struct {
	score    int
	position int
}

// computeClassReport generates a report for a class. studentsInfo is a map of
// students to their subject scores.
func (r *Resolver) computeClassReport(classID string, classSubjects []*class.Subject, studentsInfo map[string][]*student.SubjectScore) {
	var totalMaxSubjectsScore int
	subjectScoreMap := make(map[string]*subjectScoreInfo, len(classSubjects))
	for _, subjectInfo := range classSubjects {
		totalMaxSubjectsScore += subjectInfo.MaxScore
		subjectScoreMap[subjectInfo.Name] = &subjectScoreInfo{
			maxScore: subjectInfo.MaxScore,
		}
	}

	// studentReportMap is a map of studentID to student reports.
	studentReports := make([]*studentReport, 0, len(studentsInfo))
	studentReportMap := make(map[string]*student.Report, len(studentsInfo))

	// Compute max scores and subject scores for all students.
	for studentID, subjects := range studentsInfo {
		var totalScore int
		for _, subject := range subjects {
			totalScore += subject.Score

			// Group all the scores across all students for this subject.
			subjectScoreMap[subject.Name].studentScores = append(subjectScoreMap[subject.Name].studentScores, &studentSubjectScore{
				studentID: studentID,
				score:     subject.Score,
			})
		}

		report := &student.Report{
			Class: &student.ClassReport{
				TotalScore:           totalScore,
				TotalScorePercentage: float64(totalScore/totalMaxSubjectsScore) * 100,
			},
		}

		studentReportMap[studentID] = report
		studentReports = append(studentReports, &studentReport{
			studentID: studentID,
			report:    report,
		})
	}

	// subjectReportMap is a map of subject names to subject reports
	// and is used to enure students with the same subject score get the same
	// position.
	subjectReportMap := make(map[string]*subjectReport, 0)

	// Set student subject position an grade them.
	for subjectName, subject := range subjectScoreMap {
		// Sort according to highest subject scores.
		sort.SliceStable(subject.studentScores, func(i, j int) bool {
			return subject.studentScores[i].score > subject.studentScores[j].score
		})

		// Set student position and grade them.
		for positionIndex, report := range subject.studentScores {
			sr, found := subjectReportMap[subjectName]
			if !found || sr.score != report.score {
				sr = &subjectReport{
					score:    report.score,
					position: positionIndex + 1,
				}
				subjectReportMap[subjectName] = sr
			}

			subjectScorePercentage := float64(sr.score/subject.maxScore) * 100
			studentReportMap[report.studentID].Subjects = append(studentReportMap[report.studentID].Subjects, &student.SubjectReport{
				SubjectScore: &student.SubjectScore{
					Name:  subjectName,
					Score: sr.score,
				},
				Grade:    gradeTotalScorePercentage(subjectScorePercentage),
				Position: sr.position,
			})
		}
	}

	// Sort according to highest total scores.
	sort.SliceStable(studentReports, func(i, j int) bool {
		return studentReports[i].report.Class.TotalScore > studentReports[j].report.Class.TotalScore
	})

	classReport := &class.ClassReport{
		TotalStudents: len(studentReports),
	}

	// classPositionMap is a map of student total scores to a student position
	// and is used to ensure students with the same total scores get the same
	// position
	classPositionMap := make(map[int]int)

	// Set student position and grade them.
	for positionIndex, record := range studentReports {
		report := record.report.Class
		studentPosition, foundScore := classPositionMap[report.TotalScore]
		if !foundScore {
			studentPosition = positionIndex + 1
			classPositionMap[report.TotalScore] = studentPosition
		}

		studentReportMap[record.studentID].Class.Position = studentPosition
		studentReportMap[record.studentID].Class.Grade = gradeTotalScorePercentage(report.TotalScorePercentage)

		if report.TotalScore > classReport.HighestStudentScore {
			classReport.HighestStudentScore = report.TotalScore
		} else if report.TotalScore < classReport.LowestStudentScore {
			classReport.LowestStudentScore = report.TotalScore
		}
	}

	classReport.HighestStudentScoreAsPercentage = float64(classReport.HighestStudentScore/totalMaxSubjectsScore) * 100
	classReport.LowestStudentScoreAsPercentage = float64(classReport.LowestStudentScore/totalMaxSubjectsScore) * 100

	err := r.ClassRepo.SaveClassReport(classID, classReport)
	if err != nil {
		log.Printf("\nSERVER ERROR: ClassRepo.SaveClassReport %v", err.Error())
	}

	err = r.StudentRepo.SaveStudentReports(studentReportMap)
	if err != nil {
		log.Printf("\nSERVER ERROR: StudentRepo.SaveStudentReports %v", err.Error())
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
