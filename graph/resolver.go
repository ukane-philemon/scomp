package graph

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/ukane-philemon/scomp/internal/admin"
	"github.com/ukane-philemon/scomp/internal/auth"
	"github.com/ukane-philemon/scomp/internal/class"
	"github.com/ukane-philemon/scomp/internal/student"
)

type Resolver struct {
	wg sync.WaitGroup

	AdminRepository          admin.Repository
	ClassRepository          class.Repository
	StudentRepository        student.Repository
	AuthenticationRepository auth.Repository
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
			Class: &student.StudentClassReport{
				TotalScore:           totalScore,
				TotalScorePercentage: fmt.Sprintf("%.1f", float64(totalScore)/float64(totalMaxSubjectsScore)*100),
			},
		}

		studentReportMap[studentID] = report
		studentReports = append(studentReports, &studentReport{
			studentID: studentID,
			report:    report,
		})
	}

	nowUnix := time.Now().Unix()

	// Set student subject position an grade them.
	for subjectName, subject := range subjectScoreMap {
		// Sort according to highest subject scores.
		sort.SliceStable(subject.studentScores, func(i, j int) bool {
			return subject.studentScores[i].score > subject.studentScores[j].score
		})

		// Set student position and grade them.
		for positionIndex, report := range subject.studentScores {
			subjectScorePercentage := float64(report.score) / float64(subject.maxScore) * 100
			studentReportMap[report.studentID].Subjects = append(studentReportMap[report.studentID].Subjects, &student.SubjectReport{
				SubjectScore: &student.SubjectScore{
					Name:  subjectName,
					Score: report.score,
				},
				Grade:    gradeTotalScorePercentage(fmt.Sprint(subjectScorePercentage)),
				Position: positionIndex + 1,
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

	// Set student position and grade them.
	for positionIndex, record := range studentReports {
		report := record.report.Class
		studentPosition := positionIndex + 1

		if report.TotalScore > classReport.HighestStudentScore {
			classReport.HighestStudentScore = report.TotalScore
		} else if classReport.LowestStudentScore == 0 || report.TotalScore < classReport.LowestStudentScore {
			classReport.LowestStudentScore = report.TotalScore
		}

		studentReportMap[record.studentID].Class.Position = studentPosition
		studentReportMap[record.studentID].Class.Grade = gradeTotalScorePercentage(report.TotalScorePercentage)
		studentReportMap[record.studentID].GeneratedAt = fmt.Sprint(nowUnix)
	}

	classReport.HighestStudentScoreAsPercentage = fmt.Sprintf("%.1f", float64(classReport.HighestStudentScore)/float64(totalMaxSubjectsScore)*100)
	classReport.LowestStudentScoreAsPercentage = fmt.Sprintf("%1.f", float64(classReport.LowestStudentScore)/float64(totalMaxSubjectsScore)*100)
	classReport.GeneratedAt = fmt.Sprint(nowUnix)

	err := r.ClassRepository.SaveClassReport(classID, classReport)
	if err != nil {
		log.Printf("SERVER ERROR: ClassRepo.SaveClassReport %v", err.Error())
	}

	err = r.StudentRepository.SaveStudentReports(studentReportMap)
	if err != nil {
		log.Printf("SERVER ERROR: StudentRepo.SaveStudentReports %v", err.Error())
	}
}

func gradeTotalScorePercentage(totalStudentScorePercentageStr string) string {
	totalStudentScorePercentage, _ := strconv.ParseFloat(totalStudentScorePercentageStr, 64)
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
