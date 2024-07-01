package student

type Repository interface {
	// Create adds a students record. Returns db.ErrorInvalidRequest if
	// studentName already exists for classID.
	Create(classID string, studentName string, subjectScores []*SubjectScore) (string, error)
	// Student returns the students that match provided arguments.
	Student(classID string, studentID string) (*Student, error)
	// Students returns all the students that match the provided classID.
	Students(classID string) ([]*Student, error)
	// StudentScores returns a map of student ID to their subject scores.
	StudentScores(classID string) (map[string][]*SubjectScore, error)
	// SaveStudentReports saves the students report specified.
	SaveStudentReports(reports map[string]*Report) error
}
