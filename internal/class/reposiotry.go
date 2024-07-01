package class

type Repository interface {
	// Create creates a new class in the database. Returns
	// db.ErrorInvalidRequest is the provided class name matches any record in
	// the database.
	Create(className string, subjects []*Subject) (string, error)
	// Class returns information for the class that match the provided classID.
	Class(classID string) (*Class, error)
	// Classes returns information for all the classes in the database. Set
	// hasReport to filter classes by report.
	Classes(hasReport *bool) ([]*Class, error)
	// Exists checks if classID exists.
	Exists(classID string) (bool, error)
	// SaveClassReport saves a newly generated class report for the class that
	// match the provided classID.
	SaveClassReport(classID string, report *ClassReport) error
}
