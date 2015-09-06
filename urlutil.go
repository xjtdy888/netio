package netio
/*
import (
	"strings"
	"net/http"
)


/tbim/1/xhr-polling/sessionid?params
*/
/*

func URLGetTransport(r *http.Request) string {
	path := r.URL.Path
	path = strings.Replace(path, "//", "/", -1)
	spliter := strings.Split(path, "/")
	
	if len(spliter) > 0 {
		return spliter[0]
	}
	return ""
}

func URLGetSessionId(r *http.Request) string {
	path := r.URL.Path
	path = strings.Replace(path, "//", "/", -1)
	spliter := strings.Split(path, "/")


	if len(spliter) > 1 {
		return spliter[1]
	}
	return ""
}
*/