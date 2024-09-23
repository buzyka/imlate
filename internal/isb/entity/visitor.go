package entity

type Visitor struct {
	Id      int32  `json:"id"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
	Grade   int    `json:"grade"`
	Image   string `json:"image"`	
}

type VisitDetails struct {
	Visitor  *Visitor `json:"visitor"`
	Key 	 string   `json:"key"`
}