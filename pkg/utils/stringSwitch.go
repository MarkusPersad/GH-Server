package utils

func StringSwitch(s1 string,s2 string)(string,string){
	if s1 > s2 {
		return s2,s1
	} else {
		return s1,s2
	}
}