package schema


type NewMessage struct {
	Message  	string		`json:"msg" binding:"required"`
	To 			string		`json:"user" binding:"required"`
}
