// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

type ClassReport struct {
	ClassID                         string           `json:"classID"`
	ClassName                       string           `json:"className"`
	HighestStudentScore             int              `json:"highestStudentScore"`
	HighestStudentScoreAsPercentage float64          `json:"highestStudentScoreAsPercentage"`
	LowestStudentScore              int              `json:"lowestStudentScore"`
	LowestStudentScoreAsPercentage  float64          `json:"lowestStudentScoreAsPercentage"`
	StudentsReport                  []*StudentReport `json:"studentsReport"`
}

type Mutation struct {
}

type NewClass struct {
	Name               string          `json:"name"`
	ClassSubjects      []*Subject      `json:"classSubjects"`
	ClassStudentScores []*StudentScore `json:"classStudentScores"`
}

type Query struct {
}

type StudentReport struct {
	StudentID            string                  `json:"studentID"`
	StudentName          string                  `json:"studentName"`
	ClassPosition        int                     `json:"classPosition"`
	ClassGrade           string                  `json:"classGrade"`
	SubjectReport        []*StudentSubjectReport `json:"subjectReport"`
	TotalScore           int                     `json:"totalScore"`
	TotalScorePercentage float64                 `json:"totalScorePercentage"`
}

type StudentScore struct {
	StudentID    string                 `json:"studentID"`
	StudentName  string                 `json:"studentName"`
	SubjectScore []*StudentSubjectScore `json:"subjectScore"`
}

type StudentSubjectReport struct {
	Name     string `json:"name"`
	Score    int    `json:"score"`
	Grade    string `json:"grade"`
	Position int    `json:"position"`
}

type StudentSubjectScore struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type Subject struct {
	Name     string `json:"name"`
	MaxScore int    `json:"maxScore"`
}
