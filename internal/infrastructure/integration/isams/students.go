package isams

type StudentsResponse struct {
	Count      int32       `json:"count"`
	Page       int32       `json:"page"`
	PageSize   int32       `json:"pageSize"`
	Students   []Student `json:"students"`
	TotalCount int32       `json:"totalCount"`
	TotalPages int32     `json:"totalPages"`
}

type HomeAddress struct {
	ID       int64  `json:"id"`
	Address1 string `json:"address1"`
	Address2 string `json:"address2"`
	Address3 string `json:"address3"`
	Country  string `json:"country"`
	County   string `json:"county"`
	Postcode string `json:"postcode"`
	Private  bool   `json:"private"`
	Town     string `json:"town"`
}

type Student struct {
	ID                  int64         `json:"id"`
	AcademicHouse       *string       `json:"academicHouse"`
	BirthCounty         *string       `json:"birthCounty"`
	Birthplace          *string       `json:"birthplace"`
	BoardingHouse       *string       `json:"boardingHouse"`
	BoardingStatus      *string       `json:"boardingStatus"`
	DOB                 *string       `json:"dob"`           // "YYYY-MM-DD"
	EnrolmentDate       *string       `json:"enrolmentDate"` // может быть null
	EnrolmentStatus     *string       `json:"enrolmentStatus"`
	EnrolmentTerm       *string       `json:"enrolmentTerm"`
	EnrolmentYear       *int          `json:"enrolmentYear"`
	Ethnicity           *string       `json:"ethnicity"`
	FamilyID            *int64        `json:"familyId"`
	Forename            *string       `json:"forename"`
	FormGroup           *string       `json:"formGroup"`
	FullName            *string       `json:"fullName"`
	FutureSchoolID      *int64        `json:"futureSchoolId"`
	Gender              *string       `json:"gender"`
	HomeAddresses       []HomeAddress `json:"homeAddresses"` // null -> nil slice
	Initials            *string       `json:"initials"`
	IsVisaRequired      *bool         `json:"isVisaRequired"`
	LabelSalutation     *string       `json:"labelSalutation"`
	Languages           []string      `json:"languages"`
	LastUpdated         *string       `json:"lastUpdated"` // RFC3339, оставляем строкой
	LatestPhotoID       *int64        `json:"latestPhotoId"`
	LeavingDate         *string       `json:"leavingDate"`
	LeavingReason       *string       `json:"leavingReason"`
	LeavingYearGroup    *int          `json:"leavingYearGroup"`
	LetterSalutation    *string       `json:"letterSalutation"`
	Middlenames         *string       `json:"middlenames"`
	MobileNumber        *string       `json:"mobileNumber"`
	Nationalities       []string      `json:"nationalities"`
	OfficialName        *string       `json:"officialName"`
	PersonalEmail       *string       `json:"personalEmailAddress"`
	PersonGuid          string        `json:"personGuid"`
	PersonID            int64         `json:"personId"`
	PreferredName       *string       `json:"preferredName"`
	PreviousName        *string       `json:"previousName"`
	Religion            *string       `json:"religion"`
	ResidentCountry     *string       `json:"residentCountry"`
	SchoolCode          *string       `json:"schoolCode"`
	SchoolEmailAddress  *string       `json:"schoolEmailAddress"`
	SchoolID            string        `json:"schoolId"`
	Surname             *string       `json:"surname"`
	SystemStatus        *string       `json:"systemStatus"`
	Title               *string       `json:"title"`
	TutorEmployeeID     *int64        `json:"tutorEmployeeId"`
	UniquePupilNumber   *string       `json:"uniquePupilNumber"`
	YearGroup           *int          `json:"yearGroup"`
}
