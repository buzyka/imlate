package stat

const (
	TrendUP = "up"
	TrendDOWN = "down"
	TrendEQUAL = "equal"
)

type DailyGeneral struct {
	TotalVisitors 						int 	`json:"totalVisitors"`
	RegisteredVisitors 					int 	`json:"registeredVisitors"`
	PercentComparingToYesterday 		float64 `json:"percentComparingToYesterday"`
	PercentComparingToYesterdayTrend 	string 	`json:"percentComparingToYesterdayTrend"`
	PercentComparingToLastWeek 			float64 `json:"percentComparingToLastWeek"`
	PercentComparingToLastWeekTrend 	string 	`json:"percentComparingToLastWeekTrend"`
}